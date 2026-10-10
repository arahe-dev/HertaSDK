package execution

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Runtime is the shared local execution authority and resource owner.
//
// It answers exactly one question per operation invocation: "may this
// work start?" — enough resources, key available, still accepting,
// context alive. Nothing else.
type Runtime struct {
	mu sync.RWMutex
	// accepting guards admission together with mu: begin() checks it
	// under RLock and increments wg before releasing, so Shutdown (which
	// takes the write lock before flipping it) can never observe a zero
	// WaitGroup while an operation is between check and Add.
	accepting bool
	// wg counts every admitted-but-not-finished operation, whether it is
	// still waiting for admission or already executing.
	wg sync.WaitGroup

	resources map[string]*Resource
	// order is the global, deterministic acquisition order (sorted names).
	// Fixed global order ⇒ no hold-and-wait cycles ⇒ no deadlocks.
	order []string

	keys *KeyLocker

	// waitCtx is closed when shutdown begins. Operations blocked waiting
	// for a serialization key or resource capacity select on it, so
	// Shutdown wakes admission waiters immediately. Once an operation has
	// acquired everything it stops depending on waitCtx and drains.
	waitCtx    context.Context
	cancelWait context.CancelFunc

	stats atomicStats
}

// ResourceSpec declares a named finite capacity.
type ResourceSpec struct {
	Name     string
	Capacity int64
}

// NewRuntime creates a Runtime with the given resource budgets.
// Construction-time validation: duplicate names and non-positive capacities
// are errors (never fail at request time).
func NewRuntime(specs ...ResourceSpec) (*Runtime, error) {
	if len(specs) == 0 {
		return nil, errors.New("execution: runtime requires at least one resource")
	}
	resources := make(map[string]*Resource, len(specs))
	order := make([]string, 0, len(specs))
	for _, s := range specs {
		if s.Name == "" {
			return nil, errors.New("execution: resource name must not be empty")
		}
		if s.Capacity <= 0 {
			return nil, fmt.Errorf("execution: resource %q capacity must be > 0, got %d", s.Name, s.Capacity)
		}
		if _, dup := resources[s.Name]; dup {
			return nil, fmt.Errorf("execution: duplicate resource %q", s.Name)
		}
		resources[s.Name] = newResource(s.Name, s.Capacity)
		order = append(order, s.Name)
	}
	// Deterministic global acquisition order.
	sortStrings(order)

	waitCtx, cancelWait := context.WithCancel(context.Background())
	rt := &Runtime{
		accepting:  true,
		resources:  resources,
		order:      order,
		keys:       newKeyLocker(),
		waitCtx:    waitCtx,
		cancelWait: cancelWait,
	}
	return rt, nil
}

// begin admits one operation: it checks admission under RLock and
// increments the drain counter before releasing, which is the invariant
// that makes Shutdown's Wait correct (WaitGroup.Add can never race Wait).
func (rt *Runtime) begin() error {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	if !rt.accepting {
		rt.stats.rejected.Add(1)
		return ErrRuntimeStopping
	}
	rt.wg.Add(1)
	rt.stats.admitted.Add(1)
	return nil
}

// end marks one admitted operation as finished (drained).
func (rt *Runtime) end() {
	rt.wg.Done()
	rt.stats.finished.Add(1)
}

// waitDone is closed when shutdown begins; admission waiters select on it.
func (rt *Runtime) waitDone() <-chan struct{} { return rt.waitCtx.Done() }

// isStopping reports whether shutdown has begun.
func (rt *Runtime) isStopping() bool {
	select {
	case <-rt.waitDone():
		return true
	default:
		return false
	}
}

// admissionError maps a failed admission wait to the contract error:
// runtime shutdown withdraws waiters with ErrRuntimeStopping; caller
// cancellation/deadline passes through untouched.
func (rt *Runtime) admissionError(err error) error {
	if rt.isStopping() {
		return ErrRuntimeStopping
	}
	return err
}

// Shutdown stops the runtime:
//
//  1. rejects new work (Do returns ErrRuntimeStopping),
//  2. withdraws work still waiting for key/resource admission (waiters are
//     woken via the stop-admission context and fail with ErrRuntimeStopping),
//  3. drains work that already began executing (their handlers run to
//     completion; they are never force-cancelled).
//
// It returns nil once everything admitted has drained, or ctx.Err() if the
// caller's deadline expires first (drain continues in the background).
func (rt *Runtime) Shutdown(ctx context.Context) error {
	rt.mu.Lock()
	if rt.accepting {
		rt.accepting = false
		rt.cancelWait() // wake admission waiters; executing work is unaffected
	}
	rt.mu.Unlock()

	done := make(chan struct{})
	go func() {
		rt.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stats is a cheap atomic snapshot of runtime counters. Observability on
// purpose: no callbacks, no hooks, no new execution semantics.
type Stats struct {
	// Admitted: operations that passed the admission gate (includes later
	// withdrawals; they were admitted before shutdown cancelled them).
	Admitted uint64
	// Rejected: operations refused because the runtime was stopping.
	Rejected uint64
	// Started: operations that acquired everything and entered the retry
	// loop (i.e. actually began executing the handler).
	Started uint64
	// Finished: operations that completed draining (outcome produced, or
	// withdrawn while waiting).
	Finished uint64
	// Retried: extra attempts executed beyond the first.
	Retried uint64
}

func (rt *Runtime) Stats() Stats {
	return Stats{
		Admitted: rt.stats.admitted.Load(),
		Rejected: rt.stats.rejected.Load(),
		Started:  rt.stats.started.Load(),
		Finished: rt.stats.finished.Load(),
		Retried:  rt.stats.retried.Load(),
	}
}

// sortStrings is a tiny local sort to keep the package dependency-free
// beyond x/sync (sort.Slice would be fine too; this is just cheaper).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
