package hertasdk

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAuthoritativeClassifier(t *testing.T) {
	unavailable, timeout := errors.New("unavailable"), errors.New("timeout")
	for _, tc := range []struct {
		name     string
		classify func(error) Outcome
		err      error
		want     Outcome
		attempts int
	}{
		{"legacy tag", nil, Fail(Transient, timeout), Transient, 3},
		{"override tag", func(error) Outcome { return Uncertain }, Fail(Transient, timeout), Uncertain, 1},
		{"joined timeout first", func(err error) Outcome {
			if errors.Is(err, timeout) {
				return Uncertain
			}
			if errors.Is(err, unavailable) {
				return Transient
			}
			return Permanent
		}, Fail(Transient, errors.Join(unavailable, timeout)), Uncertain, 1},
		{"joined unavailable first", func(err error) Outcome {
			if errors.Is(err, unavailable) {
				return Transient
			}
			return Uncertain
		}, errors.Join(unavailable, timeout), Transient, 3},
		{"invalid verdict", func(error) Outcome { return Outcome(99) }, timeout, Permanent, 1},
		{"success verdict on error", func(error) Outcome { return Success }, timeout, Permanent, 1},
		{"nil error", func(error) Outcome { t.Fatal("classified nil"); return Permanent }, nil, Success, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := mustRuntime(t, ResourceSpec{Name: "db", Capacity: 1})
			calls := 0
			op := mustOperation(t, rt, "charge", func(context.Context, int) (int, error) {
				calls++
				return 7, tc.err
			}, Policy[int]{Effect: NonIdempotent, Classify: tc.classify,
				Retry: RetryPolicy{MaxAttempts: 3, On: map[Outcome]bool{Transient: true}}})
			_, err := op.Do(context.Background(), 0)
			if OutcomeOf(err) != tc.want || calls != tc.attempts {
				t.Fatalf("outcome=%v calls=%d, want %v/%d: %v", OutcomeOf(err), calls, tc.want, tc.attempts, err)
			}
			if tc.err != nil && !errors.Is(err, tc.err) {
				t.Fatalf("lost original error chain: %v", err)
			}
		})
	}
}

func TestAuthoritativeDeadlineAndCancellation(t *testing.T) {
	for _, verdict := range []Outcome{Transient, Permanent} {
		t.Run(verdict.String(), func(t *testing.T) {
			rt := mustRuntime(t, ResourceSpec{Name: "db", Capacity: 1})
			calls := 0
			op := mustOperation(t, rt, "charge", func(ctx context.Context, _ int) (int, error) {
				calls++
				<-ctx.Done()
				return 0, Fail(Transient, ctx.Err())
			}, Policy[int]{Effect: NonIdempotent, Timeout: time.Millisecond,
				Classify: func(error) Outcome { return verdict },
				Retry:    RetryPolicy{MaxAttempts: 3, On: map[Outcome]bool{Transient: true}}})
			_, err := op.Do(context.Background(), 0)
			if calls != 1 || OutcomeOf(err) != Uncertain || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("deadline did not win: calls=%d err=%v outcome=%v", calls, err, OutcomeOf(err))
			}
		})
	}
	t.Run("caller cancellation", func(t *testing.T) {
		rt := mustRuntime(t, ResourceSpec{Name: "db", Capacity: 1})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		op := mustOperation(t, rt, "charge", func(context.Context, int) (int, error) {
			calls++
			cancel()
			return 0, Fail(Transient, errors.New("helper error"))
		}, Policy[int]{Effect: NonIdempotent, Classify: func(error) Outcome {
			t.Fatal("classifier ran after caller cancellation")
			return Transient
		}, Retry: RetryPolicy{MaxAttempts: 3, On: map[Outcome]bool{Transient: true}}})
		_, err := op.Do(ctx, 0)
		if err != context.Canceled || calls != 1 {
			t.Fatalf("caller cancellation lost: %v, calls=%d", err, calls)
		}
	})
}

func TestClosedCalls(t *testing.T) {
	for _, declared := range []bool{false, true} {
		t.Run(map[bool]string{false: "undeclared", true: "declared"}[declared], func(t *testing.T) {
			rt := mustRuntime(t, ResourceSpec{Name: "db", Capacity: 1})
			called := 0
			child := mustOperation(t, rt, "main::child", func(context.Context, int) (int, error) {
				called++
				return 7, nil
			}, Policy[int]{Effect: Pure})
			policy := Policy[int]{Effect: Pure, Closed: true}
			if declared {
				policy.Calls = []string{"main::child"}
			}
			parent := mustOperation(t, rt, "main::parent", func(ctx context.Context, in int) (int, error) {
				return child.Do(ctx, in)
			}, policy)
			if declared {
				policy.Calls[0] = "changed after construction"
			}
			out, err := parent.Do(context.Background(), 0)
			if declared {
				if err != nil || out != 7 || called != 1 {
					t.Fatalf("declared call failed: %d %v calls=%d", out, err, called)
				}
			} else if !errors.Is(err, ErrUndeclaredCall) || OutcomeOf(err) != Permanent || called != 0 ||
				!strings.Contains(err.Error(), "main::parent -> main::child") || rt.Stats().Admitted != 1 {
				t.Fatalf("undeclared call was admitted: %v calls=%d stats=%+v", err, called, rt.Stats())
			}
		})
	}
}

func TestClosedKeyReacquired(t *testing.T) {
	for _, tc := range []struct {
		name, key string
		separate  bool
		want      error
	}{
		{"same runtime", "held", false, ErrKeyReacquired},
		{"empty key", "", false, nil},
		{"separate runtime", "held", true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := mustRuntime(t, ResourceSpec{Name: "db", Capacity: 1})
			childRT := rt
			if tc.separate {
				childRT = mustRuntime(t, ResourceSpec{Name: "db", Capacity: 1})
			}
			keyCalls, ran := 0, false
			child := mustOperation(t, childRT, "child", func(context.Context, int) (int, error) {
				ran = true
				return 0, nil
			}, Policy[int]{Effect: Pure, SerializeKey: func(int) string { keyCalls++; return tc.key }})
			middle := mustOperation(t, rt, "middle", func(ctx context.Context, in int) (int, error) {
				return child.Do(ctx, in)
			}, Policy[int]{Effect: Pure})
			parent := mustOperation(t, rt, "parent", func(ctx context.Context, in int) (int, error) {
				return middle.Do(ctx, in)
			}, Policy[int]{Effect: Pure, Closed: true, Calls: []string{"middle"},
				SerializeKey: func(int) string { return tc.key }})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_, err := parent.Do(ctx, 0)
			if !errors.Is(err, tc.want) || keyCalls != 1 || ran != (tc.want == nil) {
				t.Fatalf("err=%v ran=%v key evaluations=%d", err, ran, keyCalls)
			}
			if tc.want != nil && (OutcomeOf(err) != Permanent || !strings.Contains(err.Error(), "parent -> child")) {
				t.Fatalf("bad key diagnostic: %v", err)
			}
		})
	}
}

func TestClosedContextBoundary(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "db", Capacity: 1})
	child := mustOperation(t, rt, "child", func(context.Context, int) (int, error) { return 7, nil }, Policy[int]{Effect: Pure})
	parent := mustOperation(t, rt, "parent", func(context.Context, int) (int, error) {
		return child.Do(context.Background(), 0)
	}, Policy[int]{Effect: Pure, Closed: true})
	if out, err := parent.Do(context.Background(), 0); err != nil || out != 7 {
		t.Fatalf("detached call unexpectedly tracked: %d %v", out, err)
	}
}
