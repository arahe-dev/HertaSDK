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
			Policy[int]{Resources: []Requirement{{Name: "r", Units: 1}}, Admission: Wait})

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
				policy:  Policy[int]{},
			},
			{
				label:   "empty name",
				rt:      rt,
				name:    "",
				handler: okHandler,
				policy:  Policy[int]{},
			},
			{
				label:   "nil handler",
				rt:      rt,
				name:    "op",
				handler: nil,
				policy:  Policy[int]{},
			},
			{
				label:   "unknown resource",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy:  Policy[int]{Resources: []Requirement{{Name: "nope", Units: 1}}},
			},
			{
				label:   "zero units",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy:  Policy[int]{Resources: []Requirement{{Name: "r", Units: 0}}},
			},
			{
				label:   "negative units",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy:  Policy[int]{Resources: []Requirement{{Name: "r", Units: -1}}},
			},
			{
				label:   "over-capacity units",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy:  Policy[int]{Resources: []Requirement{{Name: "r", Units: 3}}},
			},
			{
				label:   "duplicate resource requirement",
				rt:      rt,
				name:    "op",
				handler: okHandler,
				policy: Policy[int]{Resources: []Requirement{
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
