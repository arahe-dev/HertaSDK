package hertasdk

import (
	"context"
	"errors"
	"testing"
)

var errNotSent = errors.New("not sent")

// lostAck sends on attempt 1, loses the acknowledgment and reports the
// loss as "not sent", which the boundary study showed double-commits.
func lostAck(writes *int, dispatch bool) Handler[int, int] {
	n := 0
	return func(ctx context.Context, _ int) (int, error) {
		n++
		if dispatch {
			Dispatch(ctx)
		}
		*writes++
		if n == 1 {
			return 0, Fail(Transient, errNotSent)
		}
		return *writes, nil
	}
}

func TestDispatchEvidence(t *testing.T) {
	for _, tc := range []struct {
		name            string
		track, dispatch bool
		classify        func(error) Outcome
		wantWrites      int
		want            Outcome
	}{
		{"untracked mislabel double-commits", false, true, nil, 2, Success},
		{"tracked but adapter skips Dispatch", true, false, nil, 2, Success},
		{"tracked dispatch", true, true, nil, 1, Uncertain},
		{"tracked dispatch closed classifier", true, true,
			func(error) Outcome { return Transient }, 1, Uncertain},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := newRT(t)
			writes := 0
			op, err := NewOperation(rt, "main::publish", lostAck(&writes, tc.dispatch), Policy[int]{
				Effect:        NonIdempotent,
				Retry:         RetryPolicy{MaxAttempts: 2, On: map[Outcome]bool{Transient: true}},
				Classify:      tc.classify,
				TrackDispatch: tc.track,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = op.Do(context.Background(), 0)
			if writes != tc.wantWrites || OutcomeOf(err) != tc.want {
				t.Fatalf("writes=%d outcome=%s, want %d %s", writes, OutcomeOf(err), tc.wantWrites, tc.want)
			}
			if err != nil && !errors.Is(err, errNotSent) {
				t.Fatalf("lost cause: %v", err)
			}
		})
	}
}

func TestDispatchBeforeSendStillRetries(t *testing.T) {
	rt := newRT(t)
	calls, writes := 0, 0
	op, _ := NewOperation(rt, "main::publish", func(ctx context.Context, _ int) (int, error) {
		calls++
		if calls == 1 {
			return 0, Fail(Transient, errNotSent) // refused before Dispatch
		}
		Dispatch(ctx)
		writes++
		return writes, nil
	}, Policy[int]{Effect: NonIdempotent, TrackDispatch: true,
		Retry: RetryPolicy{MaxAttempts: 2, On: map[Outcome]bool{Transient: true}}})
	if _, err := op.Do(context.Background(), 0); err != nil || calls != 2 || writes != 1 {
		t.Fatalf("err=%v calls=%d writes=%d", err, calls, writes)
	}
}

// A nested non-idempotent dispatch forbids an idempotent parent from
// retrying Uncertain; an idempotent dispatch alone does not.
func TestNestedDispatchTaintsParent(t *testing.T) {
	for _, childEffect := range []Effect{NonIdempotent, Idempotent} {
		rt := newRT(t)
		writes := 0
		child, _ := NewOperation(rt, "main::charge", func(ctx context.Context, _ int) (int, error) {
			Dispatch(ctx)
			writes++
			return 0, nil
		}, Policy[int]{Effect: childEffect, TrackDispatch: true})
		parentAttempts := 0
		parent, _ := NewOperation(rt, "main::checkout", func(ctx context.Context, in int) (int, error) {
			parentAttempts++
			if _, err := child.Do(ctx, in); err != nil {
				return 0, err
			}
			if !Dispatched(ctx) {
				t.Fatal("parent attempt not marked dispatched")
			}
			return 0, Fail(Uncertain, errors.New("receipt lost"))
		}, Policy[int]{Effect: Idempotent, TrackDispatch: true,
			Retry: RetryPolicy{MaxAttempts: 3, On: map[Outcome]bool{Uncertain: true}}})
		_, err := parent.Do(context.Background(), 0)
		want := 1
		if childEffect == Idempotent {
			want = 3
		}
		if writes != want || parentAttempts != want || OutcomeOf(err) != Uncertain {
			t.Fatalf("%s child: writes=%d attempts=%d outcome=%s", childEffect, writes, parentAttempts, OutcomeOf(err))
		}
	}
}

func TestDispatchOutsideTrackedAttempt(t *testing.T) {
	ctx := context.Background()
	Dispatch(ctx)
	if Dispatched(ctx) {
		t.Fatal("untracked context reports dispatch")
	}
}

func newRT(t *testing.T) *Runtime {
	t.Helper()
	rt, err := NewRuntime(ResourceSpec{Name: "writes", Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	return rt
}
