package execution

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"time"
)

// Operation binds a handler to a runtime under a validated, immutable
// snapshot of its Policy.
type Operation[I any, O any] struct {
	rt       *Runtime
	handler  Handler[I, O]
	key      func(I) string
	claims   []claim // ascending resource name: one global order, no deadlock
	mode     AdmissionMode
	timeout  time.Duration
	attempts int
	retry    outcomeSet // outcomes that are both requested and safe
}

// claim is n units of one budget, resolved at construction.
type claim struct {
	r *Resource
	n int64
}

// NewOperation validates policy against rt and snapshots it (CONTRACT §3).
func NewOperation[I any, O any](rt *Runtime, name string, handler Handler[I, O], policy Policy[I]) (*Operation[I, O], error) {
	if rt == nil {
		return nil, errors.New("execution: runtime must not be nil")
	}
	if handler == nil {
		return nil, errors.New("execution: handler for operation " + name + " must not be nil")
	}
	if name == "" {
		return nil, errors.New("execution: operation name must not be empty")
	}
	if err := validateEffect(name, policy.Effect); err != nil {
		return nil, err
	}
	claims, err := resolveClaims(rt, name, policy.Resources)
	if err != nil {
		return nil, err
	}
	retry, err := retryTable(name, policy.Retry.On, policy.Effect)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(claims, func(a, b claim) int { return cmp.Compare(a.r.name, b.r.name) })

	op := &Operation[I, O]{
		rt:       rt,
		handler:  handler,
		key:      policy.SerializeKey,
		claims:   claims,
		mode:     Wait,
		timeout:  max(policy.Timeout, 0),
		attempts: max(policy.Retry.MaxAttempts, 1),
		retry:    retry,
	}
	if policy.Admission == Reject {
		op.mode = Reject
	}
	return op, nil
}

// Do runs one call: gate, key, resources in name order, then the attempt
// loop. Everything acquired is released exactly once on every path,
// panics included, and a panic propagates unchanged. An uncontended call
// creates no goroutine and, without a Timeout, allocates nothing.
func (op *Operation[I, O]) Do(ctx context.Context, in I) (O, error) {
	var zero O
	rt := op.rt
	if !rt.enter() {
		return zero, ErrRuntimeStopping
	}
	defer rt.exit()

	key, err := op.admit(ctx, in)
	if err != nil {
		return zero, err
	}
	defer op.release(key, len(op.claims))
	return op.run(ctx, in)
}

// admit acquires the call's key, then its claims (CONTRACT §4). A call
// waiting for its key holds no units; a call waiting for a later claim
// holds the earlier ones (§4.3). On failure it holds nothing.
func (op *Operation[I, O]) admit(ctx context.Context, in I) (*keyEntry, error) {
	rt := op.rt
	k := ""
	if op.key != nil {
		k = op.key(in)
	}
	// Only a live caller takes a key or waits for units (Amendment 1): a
	// done caller would hold capacity it cannot use (A3). Reject claims do
	// not consult ctx, so an unkeyed Reject call still reports overload.
	if k != "" || op.mode == Wait {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	var key *keyEntry
	if k != "" {
		var ok bool
		if key, ok = rt.keys.lock(ctx, k, op.mode); !ok {
			return nil, rt.refused(ctx, op.mode)
		}
	}
	for i, c := range op.claims {
		var ok bool
		if op.mode == Reject {
			ok = c.r.tryAcquire(c.n)
		} else {
			ok = c.r.acquire(ctx, c.n)
		}
		if !ok {
			op.release(key, i)
			return nil, rt.refused(ctx, op.mode)
		}
	}
	return key, nil
}

// release returns the first n claims in reverse order, then the key.
func (op *Operation[I, O]) release(key *keyEntry, n int) {
	for i := n - 1; i >= 0; i-- {
		op.claims[i].r.release(op.claims[i].n)
	}
	op.rt.keys.unlock(key)
}
