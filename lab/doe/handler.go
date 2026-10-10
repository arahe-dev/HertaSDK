package doe

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// errInjected is the handler's injected failure, tagged Transient.
var errInjected = errors.New("doe: injected failure")

// slot is one caller's scratch space. The Op input is the slot index, so the
// handler finds its call's key and timing without allocating. A slot is
// written by its own caller and by the handler running on its behalf, never
// concurrently. Padded to a cache line so callers do not share lines.
type slot struct {
	entered int64  // first handler entry (run clock); 0 = never entered
	done    uint64 // calls completed inside the measurement window
	key     int32  // -1 when unkeyed
	units   int32  // units claimed per budget
	_       [40]byte
}

// monitor checks, inside the handler, the two safety properties the runtime
// promises: per-budget capacity and per-key mutual exclusion. It counts what
// running handlers hold, a lower bound of what the runtime has granted, so
// any violation it sees is real.
type monitor struct {
	capacity int64
	inUse    []atomic.Int64
	holders  []atomic.Int32
	capViol  atomic.Uint64
	keyViol  atomic.Uint64
}

func (m *monitor) enter(s *slot) {
	for i := range m.inUse {
		if m.inUse[i].Add(int64(s.units)) > m.capacity {
			m.capViol.Add(1)
		}
	}
	if s.key >= 0 && m.holders[s.key].Add(1) > 1 {
		m.keyViol.Add(1)
	}
}

func (m *monitor) exit(s *slot) {
	for i := range m.inUse {
		m.inUse[i].Add(-int64(s.units))
	}
	if s.key >= 0 {
		m.holders[s.key].Add(-1)
	}
}

// handle is the workload's business logic: record first entry, hold the
// claimed units for the service time, maybe fail.
func (e *engine) handle(ctx context.Context, in int) (int, error) {
	s := &e.slots[in]
	if s.entered == 0 {
		s.entered = e.now()
	}
	e.mon.enter(s)
	defer e.mon.exit(s)

	d := e.w.Service
	if e.w.ExpService {
		d = time.Duration(rand.ExpFloat64() * float64(d))
	}
	if err := serve(ctx, d); err != nil {
		return 0, err
	}
	if e.w.FailRate > 0 && rand.Float64() < e.w.FailRate {
		return 0, e.im.Fail(impl.Transient, errInjected)
	}
	return in, nil
}

// serve occupies the handler for d, returning early with ctx.Err() if ctx
// ends. Below 1ms it busy-spins: Windows timer resolution would otherwise
// round short sleeps up and confound service time with the OS.
func serve(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	done := ctx.Done()
	if d >= time.Millisecond {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-t.C:
			return nil
		case <-done:
			return ctx.Err()
		}
	}
	for end := nanotime() + int64(d); nanotime() < end; {
		select {
		case <-done:
			return ctx.Err()
		default:
		}
	}
	return nil
}
