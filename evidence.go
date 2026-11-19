package hertasdk

import (
	"context"
	"sync/atomic"
)

type dispatchKey struct{}

// dispatchState is one attempt of a TrackDispatch operation. Dispatch marks
// it and every tracked ancestor attempt: an effect sent by a nested call was
// sent by each attempt that contains it.
type dispatchState struct {
	parent        *dispatchState
	effect        Effect
	dispatched    atomic.Bool
	nonIdempotent atomic.Bool
}

// Dispatch records that the current attempt may now reach the provider.
// Adapters call it immediately before the request can leave the process.
// After Dispatch, a Transient verdict ("not applied") is no longer evidence
// and becomes Uncertain; a non-idempotent dispatch also forbids enclosing
// attempts from retrying Uncertain. Outside a TrackDispatch attempt it is
// a no-op. Herta cannot detect a send that skips Dispatch.
func Dispatch(ctx context.Context) {
	cur, _ := ctx.Value(dispatchKey{}).(*dispatchState)
	nonIdempotent := cur != nil && cur.effect == NonIdempotent
	for s := cur; s != nil; s = s.parent {
		s.dispatched.Store(true)
		if nonIdempotent {
			s.nonIdempotent.Store(true)
		}
	}
}

// Dispatched reports whether the current tracked attempt has dispatched.
func Dispatched(ctx context.Context) bool {
	s, _ := ctx.Value(dispatchKey{}).(*dispatchState)
	return s != nil && s.dispatched.Load()
}

// track installs a fresh attempt state, or returns ctx unchanged.
func (op *Operation[I, O]) track(ctx context.Context) (context.Context, *dispatchState) {
	if !op.trackDispatch {
		return ctx, nil
	}
	parent, _ := ctx.Value(dispatchKey{}).(*dispatchState)
	s := &dispatchState{parent: parent, effect: op.effect}
	return context.WithValue(ctx, dispatchKey{}, s), s
}

// evidence downgrades o by what the attempt sent. retry is false when a
// dispatched non-idempotent effect makes an Uncertain retry unsafe.
func (s *dispatchState) evidence(o Outcome) (_ Outcome, retry bool) {
	if s == nil {
		return o, true
	}
	if o == Transient && s.dispatched.Load() {
		o = Uncertain
	}
	return o, o != Uncertain || !s.nonIdempotent.Load()
}
