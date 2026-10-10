package execution

import (
	"context"
	"errors"
	"sync/atomic"
)

// atomicStats backs Runtime.Stats — plain atomics, no hooks, no new
// execution semantics (no callbacks that could block/panic/reenter).
type atomicStats struct {
	admitted atomic.Uint64
	rejected atomic.Uint64
	started  atomic.Uint64
	finished atomic.Uint64
	retried  atomic.Uint64
}

// Operation is a typed unit of application work bound to a Runtime and a
// Policy. Its single public action is Do.
type Operation[I any, O any] struct {
	runtime *Runtime
	name    string
	handler Handler[I, O]
	policy  Policy[I]

	// resources is the policy's requirement list normalized once at
	// construction into the runtime's global acquisition order
	// (deterministic ⇒ no hold-and-wait cycles ⇒ deadlock-free).
	resources []Requirement
}

// NewOperation binds a handler and policy to the runtime. All contract
// validation happens here — an invalid Policy fails construction, never a
// production request.
func NewOperation[I any, O any](
	rt *Runtime,
	name string,
	handler Handler[I, O],
	policy Policy[I],
) (*Operation[I, O], error) {
	if rt == nil {
		return nil, errors.New("execution: runtime must not be nil")
	}
	if handler == nil {
		return nil, errors.New("execution: handler for operation " + name + " must not be nil")
	}
	if err := validateOperation(rt, name, policy); err != nil {
		return nil, err
	}

	// Normalize requirements into the runtime's global order once.
	sorted := make([]Requirement, len(policy.Resources))
	copy(sorted, policy.Resources)
	sortByRuntimeOrder(sorted, rt.order)

	return &Operation[I, O]{
		runtime:   rt,
		name:      name,
		handler:   handler,
		policy:    policy,
		resources: sorted,
	}, nil
}

// Do executes the operation:
//
//	Do(ctx, input)
//	  │
//	  ├─ rt.begin()            admission gate; admitted work is counted
//	  │
//	  ├─ admission ctx         caller ctx + runtime stop signal
//	  │
//	  ├─ acquire key           BEFORE resources: a queued same-key request
//	  │                        holds zero resource budget
//	  ├─ acquire resources     global order; full rollback on partial failure
//	  │
//	  │    operation is now EXECUTING
//	  │
//	  ├─ retry loop            fresh cooperative timeout per attempt;
//	  │                        classify; retry only if policy allows AND
//	  │                        semantics are safe
//	  │
//	  ├─ release resources     (defers, reverse order)
//	  ├─ release key
//	  └─ rt.end()              drained
//
// Timeout contract: each attempt gets a fresh deadline derived from the
// CALLER's context and propagated to the handler. It is cooperative —
// Herta does not forcibly terminate handlers that ignore their context.
// Caller cancellation aborts the retry loop immediately. Panics: all
// resources/keys are released, then the panic propagates — Herta never
// swallows programmer bugs.
func (op *Operation[I, O]) Do(ctx context.Context, input I) (O, error) {
	var zero O

	// 1. Admission gate (lock-gated; wg.Add cannot race Shutdown's Wait).
	if err := op.runtime.begin(); err != nil {
		return zero, err // ErrRuntimeStopping
	}
	// Exactly one end() for every successful begin(), on every path.
	defer op.runtime.end()

	// 2. Admission context = caller ctx + runtime stop signal. Used ONLY
	//    for waiting (key/resources) — never handed to the handler, so
	//    executing business logic is never force-cancelled by shutdown.
	//    The watcher goroutine lives for this invocation only; cancel is
	//    idempotent, so watcher and defer cannot conflict.
	admissionCtx, cancelAdmission := context.WithCancel(ctx)
	defer cancelAdmission()
	watchStop := make(chan struct{})
	defer close(watchStop)
	go func() {
		select {
		case <-op.runtime.waitDone():
			cancelAdmission() // wake key/resource waiters on shutdown
		case <-watchStop:
		}
	}()

	// 3. Serialization key first (nil fn or empty key ⇒ no serialization).
	releaseKey, err := op.acquireKey(admissionCtx, input)
	if err != nil {
		return zero, op.runtime.admissionError(err) // stopping, or caller ctx
	}
	defer releaseKey()

	// 4. Resources in global order with complete rollback on failure.
	releaseResources, err := op.acquireResources(admissionCtx)
	if err != nil {
		return zero, op.runtime.admissionError(err) // Throttled/stopping/ctx
	}
	defer releaseResources()

	// Now EXECUTING: the handler depends only on the caller's ctx.
	op.runtime.stats.started.Add(1)

	// 5. Retry loop; releases and end() run via the defers above.
	return op.execute(ctx, input)
}

// execute runs the retry loop with a fresh cooperative timeout per attempt.
func (op *Operation[I, O]) execute(ctx context.Context, input I) (O, error) {
	var zero O
	maxAttempts := op.policy.Retry.attempts()

	var result O
	var lastErr error
	var lastDeadlineExpired bool

	// finish wraps the final error when Herta itself made the semantic
	// call: an unclassified deadline hit mid-attempt is conservatively
	// promoted to Uncertain (adapter said nothing; the attempt may or may
	// not have taken effect). Already-classified and plain errors pass
	// through untouched.
	finish := func() (O, error) {
		var f *Failure
		if lastDeadlineExpired && OutcomeOf(lastErr) == Permanent && !errors.As(lastErr, &f) {
			return zero, Fail(Uncertain, lastErr)
		}
		return zero, lastErr
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Caller gave up entirely: abort regardless of retry policy and
		// report the cancellation itself (the last handler error is moot —
		// the caller owns the reason for stopping).
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		if attempt > 1 {
			op.runtime.stats.retried.Add(1)
		}

		attemptCtx := ctx
		lastDeadlineExpired = false
		if op.policy.Timeout > 0 {
			var cancel context.CancelFunc
			attemptCtx, cancel = context.WithTimeout(ctx, op.policy.Timeout)
			result, lastErr = op.handler(attemptCtx, input)
			// Read deadline state BEFORE cancel: after cancel(), Err()
			// reports Canceled, not DeadlineExceeded.
			lastDeadlineExpired = attemptCtx.Err() == context.DeadlineExceeded
			cancel()
		} else {
			result, lastErr = op.handler(ctx, input)
		}

		if lastErr == nil {
			return result, nil
		}

		outcome := classify(lastErr, lastDeadlineExpired)

		// Belt-and-braces: construction validation guarantees this, but the
		// retry loop enforces the safety contract independently.
		if !op.policy.Retry.mayRetry(outcome) || !isSafeRetry(op.policy.Effect, outcome) {
			return finish()
		}
		// Retry: next attempt gets a fresh timeout. No backoff in V0.
	}

	return finish()
}

// acquireKey derives and locks the serialization key (or returns a no-op).
func (op *Operation[I, O]) acquireKey(ctx context.Context, input I) (func(), error) {
	if op.policy.SerializeKey == nil {
		return noopRelease, nil
	}
	key := op.policy.SerializeKey(input)
	if key == "" {
		return noopRelease, nil // empty key ⇒ no serialization
	}
	unlock, err := op.runtime.keys.acquire(ctx, key)
	if err != nil {
		return nil, err // caller ctx cancelled, or runtime stopping
	}
	return unlock, nil
}

// acquireResources takes all required resources in the runtime's global
// order with full rollback on partial failure, honoring admission mode.
func (op *Operation[I, O]) acquireResources(ctx context.Context) (func(), error) {
	acquired := make([]*Resource, 0, len(op.resources))
	units := make([]int64, 0, len(op.resources))

	releaseAll := func() {
		// Reverse order; nothing here can panic (semaphore Release on
		// held units is safe), so the defer chain in Do stays intact.
		for i := len(acquired) - 1; i >= 0; i-- {
			acquired[i].release(units[i])
		}
	}

	for _, req := range op.resources {
		res := op.runtime.resources[req.Name]
		switch op.policy.Admission {
		case Reject:
			if !res.tryAcquire(req.Units) {
				releaseAll()
				op.runtime.stats.rejected.Add(1)
				return nil, Fail(Throttled, ErrOverloaded)
			}
		default: // Wait
			if err := res.acquire(ctx, req.Units); err != nil {
				releaseAll()
				return nil, err
			}
		}
		acquired = append(acquired, res)
		units = append(units, req.Units)
	}
	return releaseAll, nil
}

func noopRelease() {}

// sortByRuntimeOrder sorts requirements into the runtime's global order.
func sortByRuntimeOrder(reqs []Requirement, order []string) {
	pos := make(map[string]int, len(order))
	for i, n := range order {
		pos[n] = i
	}
	for i := 1; i < len(reqs); i++ {
		for j := i; j > 0 && pos[reqs[j].Name] < pos[reqs[j-1].Name]; j-- {
			reqs[j], reqs[j-1] = reqs[j-1], reqs[j]
		}
	}
}
