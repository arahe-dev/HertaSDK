package execution

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestRaceHammerHeavy is the extended Do-vs-Shutdown race proof: 200
// iterations of 16 goroutines x 50 Do calls each, with Shutdown racing
// mid-run. Every iteration must preserve the drain invariants:
// Admitted == Finished (exactly one end() per begin()) and
// Started <= Admitted (nothing executes without admission).
func TestRaceHammerHeavy(t *testing.T) {
	for iter := 0; iter < 200; iter++ {
		rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 4})
		op := mustOperation(t, rt, "spin",
			func(ctx context.Context, _ int) (int, error) { return iter, nil },
			Policy[int]{Effect: Pure, Resources: []Requirement{{Name: "r", Units: 1}}, Admission: Wait})

		var wg sync.WaitGroup
		for g := 0; g < 16; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 50; i++ {
					_, _ = op.Do(context.Background(), i) // may legally fail with stopping
				}
			}()
		}
		time.Sleep(time.Duration(iter%3) * time.Millisecond)
		_ = rt.Shutdown(context.Background())
		wg.Wait()

		s := rt.Stats()
		if s.Admitted != s.Finished {
			t.Fatalf("iter %d: Admitted %d != Finished %d — drain accounting broken",
				iter, s.Admitted, s.Finished)
		}
		if s.Started > s.Admitted {
			t.Fatalf("iter %d: Started %d > Admitted %d", iter, s.Started, s.Admitted)
		}
	}
}

// TestConstructionValidationTable exhaustively covers every NewRuntime and
// NewOperation construction-time error branch. Each case must fail
// construction; the unsafe-retry case must additionally match the
// ErrUnsafeRetry sentinel.
func TestConstructionValidationTable(t *testing.T) {
	okHandler := func(ctx context.Context, i int) (int, error) { return i, nil }

	t.Run("runtime", func(t *testing.T) {
		cases := []struct {
			label string
			specs []ResourceSpec
		}{
			{"zero specs", nil},
			{"empty name", []ResourceSpec{{Name: "", Capacity: 1}}},
			{"capacity 0", []ResourceSpec{{Name: "a", Capacity: 0}}},
			{"negative capacity", []ResourceSpec{{Name: "a", Capacity: -4}}},
			{"duplicate name", []ResourceSpec{{Name: "a", Capacity: 1}, {Name: "a", Capacity: 2}}},
		}
		for _, tc := range cases {
			t.Run(tc.label, func(t *testing.T) {
				if _, err := NewRuntime(tc.specs...); err == nil {
					t.Fatalf("%s: want construction error, got nil", tc.label)
				}
			})
		}
	})

	t.Run("operation", func(t *testing.T) {
		rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 2})

		cases := []struct {
			label    string
			rt       *Runtime
			name     string
			handler  Handler[int, int]
			policy   Policy[int]
			sentinel error
		}{
			{
				label:   "nil runtime",
				rt:      nil,
				name:    "op",
				handler: okHandler,
				policy:  Policy[int]{Effect: Pure},
			},
			{
				label:   "empty name",
				rt:      rt,
				name:    "",
				handler: okHandler,
				policy:  Policy[int]{Effect: Pure},
			},
			{
				label:   "nil handler",
				rt:      rt,
				name:    "op",
				handler: nil,
				policy:  Policy[int]{Effect: Pure},
			},
			{
				label:   "unknown resource",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy:  Policy[int]{Effect: Pure, Resources: []Requirement{{Name: "nope", Units: 1}}},
			},
			{
				label:   "zero units",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy:  Policy[int]{Effect: Pure, Resources: []Requirement{{Name: "r", Units: 0}}},
			},
			{
				label:   "negative units",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy:  Policy[int]{Effect: Pure, Resources: []Requirement{{Name: "r", Units: -1}}},
			},
			{
				label:   "over-capacity units",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy:  Policy[int]{Effect: Pure, Resources: []Requirement{{Name: "r", Units: 3}}},
			},
			{
				label:   "duplicate resource requirement",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy: Policy[int]{Effect: Pure, Resources: []Requirement{
					{Name: "r", Units: 1},
					{Name: "r", Units: 1},
				}},
			},
			{
				label:   "NonIdempotent retry-on-Uncertain",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy: Policy[int]{
					Effect: NonIdempotent,
					Retry:  RetryPolicy{MaxAttempts: 2, On: map[Outcome]bool{Uncertain: true}},
				},
				sentinel: ErrUnsafeRetry,
			},
			{
				// v0.2.0: omission is a contract error, not silent Pure.
				label:    "Effect omitted (EffectUnknown)",
				rt:       rt,
				name:     "op",
				handler:  okHandler,
				policy:   Policy[int]{},
				sentinel: ErrEffectRequired,
			},
			{
				label:    "invalid Effect value",
				rt:       rt,
				name:     "op",
				handler:  okHandler,
				policy:   Policy[int]{Effect: Effect(42)},
				sentinel: ErrInvalidEffect,
			},
			{
				// v0.2.0: Throttled is observable but not retryable in V0
				// (no backoff semantics); requesting it is a construction
				// error instead of silently ignored policy.
				label:   "retry on Throttled",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy: Policy[int]{
					Effect: Idempotent,
					Retry:  RetryPolicy{MaxAttempts: 2, On: map[Outcome]bool{Throttled: true}},
				},
				sentinel: ErrInvalidRetryPolicy,
			},
			{
				label:   "retry on Success",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy: Policy[int]{
					Effect: Idempotent,
					Retry:  RetryPolicy{MaxAttempts: 2, On: map[Outcome]bool{Success: true}},
				},
				sentinel: ErrInvalidRetryPolicy,
			},
			{
				label:   "retry on Permanent",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy: Policy[int]{
					Effect: Idempotent,
					Retry:  RetryPolicy{MaxAttempts: 2, On: map[Outcome]bool{Permanent: true}},
				},
				sentinel: ErrInvalidRetryPolicy,
			},
			{
				label:   "invalid Outcome value in Retry.On",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy: Policy[int]{
					Effect: Idempotent,
					Retry:  RetryPolicy{MaxAttempts: 2, On: map[Outcome]bool{Outcome(99): true}},
				},
				sentinel: ErrInvalidRetryPolicy,
			},
		}
		for _, tc := range cases {
			t.Run(tc.label, func(t *testing.T) {
				_, err := NewOperation(tc.rt, tc.name, tc.handler, tc.policy)
				if err == nil {
					t.Fatalf("%s: want construction error, got nil", tc.label)
				}
				if tc.sentinel != nil && !errors.Is(err, tc.sentinel) {
					t.Fatalf("%s: want error matching %v, got %v", tc.label, tc.sentinel, err)
				}
			})
		}
	})
}

// TestFailNormalizesInvalidOutcome proves the v0.2.0 normalization: an
// invalid Outcome can never be manufactured into a Failure. A "successful
// failure" (Fail(Success, err)) is a caller bug and is normalized to
// Permanent (never retried); out-of-range Outcome values likewise. The
// cause stays reachable via Unwrap; legitimate verdicts, Fail(_, nil)==nil,
// and OutcomeOf(nil)==Success are untouched.
func TestFailNormalizesInvalidOutcome(t *testing.T) {
	boom := errors.New("boom")

	successful := Fail(Success, boom)
	if got := OutcomeOf(successful); got != Permanent {
		t.Fatalf("Fail(Success, err) must normalize to Permanent, got %v", got)
	}
	if !errors.Is(successful, boom) {
		t.Fatal("normalization must preserve the cause via Unwrap")
	}

	invalid := Fail(Outcome(42), boom)
	if got := OutcomeOf(invalid); got != Permanent {
		t.Fatalf("Fail(Outcome(42), err) must normalize to Permanent, got %v", got)
	}
	if !errors.Is(invalid, boom) {
		t.Fatal("normalization must preserve the cause via Unwrap")
	}

	if got := OutcomeOf(Fail(Transient, boom)); got != Transient {
		t.Fatalf("valid verdict must be recorded verbatim, got %v", got)
	}
	if Fail(Transient, nil) != nil {
		t.Fatal("Fail(_, nil) must remain nil")
	}
	if OutcomeOf(nil) != Success {
		t.Fatal("OutcomeOf(nil) must remain Success")
	}
	if EffectUnknown.String() != "EffectUnknown" || Pure.String() != "Pure" || NonIdempotent.String() != "NonIdempotent" {
		t.Fatal("Effect.String rendering broken")
	}
}
