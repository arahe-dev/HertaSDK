package doe

import (
	"context"
	"math"
	"math/rand/v2"
	"runtime"
	"sync"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// openQueue bounds arrivals waiting for a free caller in open-loop mode;
// arrivals beyond it are dropped and counted (DropRate).
const openQueue = 4096

// launch starts the callers (and, in open-loop mode, the Poisson
// dispatcher). It returns their WaitGroup and how many goroutines it began.
func (e *engine) launch() (*sync.WaitGroup, int) {
	var wg sync.WaitGroup
	var arrivals chan int64
	n := e.w.Clients
	if e.w.Rate > 0 {
		arrivals = make(chan int64, openQueue)
		wg.Go(func() { e.dispatch(arrivals) })
		n++
	}
	for id := range e.w.Clients {
		wg.Go(func() { e.caller(id, arrivals) })
	}
	return &wg, n
}

// caller is one client: closed loop (next call when the last returns) when
// arrivals is nil, else one server of the open-loop queue.
func (e *engine) caller(id int, arrivals <-chan int64) {
	rng := rand.New(rand.NewPCG(e.w.Seed, uint64(id)))
	pick := func() int32 { return -1 }
	switch {
	case e.w.Keys > 0 && e.w.ZipfS > 1:
		z := rand.NewZipf(rng, e.w.ZipfS, 1, uint64(e.w.Keys-1))
		pick = func() int32 { return int32(z.Uint64()) }
	case e.w.Keys > 0:
		pick = func() int32 { return int32(rng.IntN(e.w.Keys)) }
	}
	if arrivals == nil {
		for !e.stop.Load() && !e.call(id, rng, pick, 0) {
		}
		return
	}
	for t0 := range arrivals {
		if e.call(id, rng, pick, t0) {
			return
		}
	}
}

// call issues one Do on behalf of caller id. t0 is the scheduled arrival in
// open-loop mode, 0 in closed-loop mode (arrival = Do start). It reports
// whether the runtime is stopping: a caller then stops, as a real client
// would, instead of spinning on refusals and starving the drain of CPU.
func (e *engine) call(id int, rng *rand.Rand, pick func() int32, t0 int64) (stopping bool) {
	s := &e.slots[id]
	s.key, s.units, s.entered = pick(), 1, 0
	op := e.ops[0]
	if e.w.HeavyFrac > 0 && rng.Float64() < e.w.HeavyFrac {
		s.units, op = int32(e.w.heavyUnits()), e.ops[1]
	}
	ctx := context.Background()
	if e.w.CancelRate > 0 && rng.Float64() < e.w.CancelRate {
		span := float64(2*e.w.Service + 50*time.Microsecond)
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(rng.Float64()*span))
		defer cancel()
	}
	tDo := e.now()
	if t0 == 0 {
		t0 = tDo
	}
	_, err := op.Do(ctx, id)
	t1 := e.now()
	if e.inWindow(t0, t1) {
		s.done++
		e.rec.call(id, e.im, t0, tDo, s.entered, t1, err)
	}
	return err != nil && e.im.Classify(err) == impl.ClassStopping
}

// dispatch emits Poisson arrivals at Rate until stop, never blocking: a full
// queue drops the arrival. It polls rather than sleeps, because Windows
// timers cannot schedule sub-millisecond gaps.
func (e *engine) dispatch(out chan<- int64) {
	defer close(out)
	rng := rand.New(rand.NewPCG(e.w.Seed, math.MaxUint64))
	gap := 1e9 / e.w.Rate
	next := e.now()
	for !e.stop.Load() {
		for now := e.now(); next <= now; next += max(1, int64(rng.ExpFloat64()*gap)) {
			select {
			case out <- next:
			default:
				if e.inWindow(next, next) {
					e.dropped.Add(1)
				}
			}
		}
		runtime.Gosched()
	}
}
