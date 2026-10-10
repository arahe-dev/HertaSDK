package hertasdk

import (
	"context"
	"sync/atomic"
)

// gate word layout: the stopping and drained bits above an in-flight count.
// Admission and drain are decided by this one word, without locks.
const (
	stopping = 1 << 63
	drained  = 1 << 62
)

// Runtime owns the resource budgets, the key space, and the admission gate
// shared by every Operation bound to it.
type Runtime struct {
	gate  atomic.Uint64
	stats counters

	done    chan struct{} // closed once stopping with nothing in flight
	keys    KeyLocker
	budgets map[string]*Resource // read-only after NewRuntime
}

// counters back Stats. Each is atomic and monotonic.
type counters struct {
	admitted, rejected, started, finished, retried atomic.Uint64
}

// Stats is a snapshot of the runtime counters.
type Stats struct {
	// Admitted counts calls that passed the gate.
	Admitted uint64
	// Rejected counts calls refused because the runtime was stopping, plus
	// Reject-mode overloads.
	Rejected uint64
	// Started counts calls whose handler ran at least once.
	Started uint64
	// Finished counts admitted calls that have fully left Do, withdrawals
	// included.
	Finished uint64
	// Retried counts attempts beyond the first.
	Retried uint64
}

// NewRuntime creates a runtime with the given budgets. It rejects no
// budgets, an empty or duplicate name, and a capacity ≤ 0.
func NewRuntime(specs ...ResourceSpec) (*Runtime, error) {
	if err := validateSpecs(specs); err != nil {
		return nil, err
	}
	rt := &Runtime{
		done:    make(chan struct{}),
		budgets: make(map[string]*Resource, len(specs)),
	}
	for _, s := range specs {
		rt.budgets[s.Name] = newResource(s.Name, s.Capacity)
	}
	return rt, nil
}

// enter admits one call unless the runtime is stopping. An admitted call
// is in flight until exit.
func (rt *Runtime) enter() bool {
	if rt.gate.Add(1)&stopping != 0 {
		rt.leave()
		rt.stats.rejected.Add(1)
		return false
	}
	rt.stats.admitted.Add(1)
	return true
}

// exit ends an admitted call. Finished is counted before the call stops
// being in flight, so a nil Shutdown always sees Finished == Admitted (Δ2).
func (rt *Runtime) exit() {
	rt.stats.finished.Add(1)
	rt.leave()
}

func (rt *Runtime) leave() {
	if rt.gate.Add(^uint64(0)) == stopping {
		rt.markDrained()
	}
}

// markDrained closes done exactly once, when the gate is stopping with
// nothing in flight. A refused call may bump the count between a drain and
// this CAS; it then brings the word back to stopping and retries here.
func (rt *Runtime) markDrained() {
	if rt.gate.CompareAndSwap(stopping, stopping|drained) {
		close(rt.done)
	}
}

// Shutdown stops admitting, withdraws every blocked waiter, and waits for
// admitted calls to finish. Executing handlers are never cancelled. It
// returns nil once drained, or ctx.Err() if ctx ends first; draining
// continues either way. It is idempotent, safe to call concurrently, and
// starts no goroutines (Δ8). Drained wins over an expired ctx.
func (rt *Runtime) Shutdown(ctx context.Context) error {
	if rt.gate.Or(stopping)&stopping == 0 {
		for _, r := range rt.budgets {
			r.close()
		}
		rt.keys.close()
		rt.markDrained()
	}
	if isClosed(rt.done) {
		return nil
	}
	select {
	case <-rt.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stats returns the counters. Each call's increments are ordered admitted
// before started before finished, and they are read in reverse, so a
// snapshot never shows Started or Finished above Admitted.
func (rt *Runtime) Stats() Stats {
	var out Stats
	s := &rt.stats
	out.Finished = s.finished.Load()
	out.Started = s.started.Load()
	out.Admitted = s.admitted.Load()
	out.Rejected = s.rejected.Load()
	out.Retried = s.retried.Load()
	return out
}
