package execution

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Contract points that are not deltas but that the v0.2 suite leaves
// unproven (its surviving mutants M05, M16, M48, M54, M11; HERTA_REVIEW).

func TestContractShutdownWaitsForDrain(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	release := newGate()
	op := mustOperation(t, rt, "stuck",
		func(context.Context, int) (int, error) { release.wait(); return 0, nil },
		Policy[int]{Effect: Pure})
	go func() { _, _ = op.Do(context.Background(), 0) }()
	waitFor(t, func() bool { return rt.Stats().Started == 1 })

	expired, cancel := context.WithCancel(context.Background())
	cancel()
	if err := rt.Shutdown(expired); !errors.Is(err, context.Canceled) {
		t.Fatalf("Shutdown with an ended ctx while draining: %v, want Canceled", err)
	}
	shut := make(chan error, 1)
	go func() { shut <- rt.Shutdown(context.Background()) }()
	select {
	case err := <-shut:
		t.Fatalf("Shutdown returned %v while a handler was running", err)
	case <-time.After(50 * time.Millisecond):
	}
	release.open()
	if err := <-shut; err != nil {
		t.Fatalf("Shutdown after drain: %v", err)
	}
}

func TestContractIdleShutdownBeatsExpiredContext(t *testing.T) {
	expired, cancel := context.WithCancel(context.Background())
	cancel()
	for range 1000 {
		rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
		if err := rt.Shutdown(expired); err != nil {
			t.Fatalf("idle Shutdown with an ended ctx: %v, want nil", err)
		}
	}
}

func TestContractEveryAttemptGetsAFreshDeadline(t *testing.T) {
	const timeout = 30 * time.Millisecond
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	var spans []time.Duration
	op := mustOperation(t, rt, "slow",
		func(ctx context.Context, _ int) (int, error) {
			start := time.Now()
			<-ctx.Done()
			spans = append(spans, time.Since(start))
			return 0, ctx.Err()
		},
		Policy[int]{Effect: Idempotent, Timeout: timeout,
			Retry: RetryPolicy{MaxAttempts: 3, On: map[Outcome]bool{Uncertain: true}}})
	_, _ = op.Do(context.Background(), 0)
	if len(spans) != 3 {
		t.Fatalf("%d attempts, want 3", len(spans))
	}
	for i, d := range spans {
		if d < timeout*8/10 {
			t.Fatalf("attempt %d ran %v; a fresh deadline gives ≈%v", i+1, d, timeout)
		}
	}
}

func TestContractRetryNeedsOn(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	calls := 0
	op := mustOperation(t, rt, "op",
		func(context.Context, int) (int, error) { calls++; return 0, Fail(Transient, errors.New("x")) },
		Policy[int]{Effect: Pure, Retry: RetryPolicy{MaxAttempts: 5}})
	_, _ = op.Do(context.Background(), 0)
	if calls != 1 {
		t.Fatalf("MaxAttempts without On retried: %d calls", calls)
	}
}

func TestContractValidationIsDeterministic(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	h := func(context.Context, int) (int, error) { return 0, nil }
	on := map[Outcome]bool{Throttled: true, Permanent: true, Outcome(99): true, Outcome(-3): true, Uncertain: true}
	_, first := NewOperation(rt, "op", h, Policy[int]{Effect: NonIdempotent, Retry: RetryPolicy{On: on}})
	if !errors.Is(first, ErrInvalidRetryPolicy) {
		t.Fatalf("invalid entries must be reported before unsafe ones, got %v", first)
	}
	for range 200 {
		if _, err := NewOperation(rt, "op", h, Policy[int]{Effect: NonIdempotent, Retry: RetryPolicy{On: on}}); err.Error() != first.Error() {
			t.Fatalf("validation varies: %q then %q", first, err)
		}
	}
}

func TestContractLateResultIsReturned(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	op := mustOperation(t, rt, "late",
		func(ctx context.Context, _ int) (int, error) { <-ctx.Done(); return 42, nil },
		Policy[int]{Effect: Pure, Timeout: time.Millisecond})
	if got, err := op.Do(context.Background(), 0); got != 42 || err != nil {
		t.Fatalf("late result: (%d, %v), want (42, nil)", got, err)
	}
}

func TestContractPanicNilPropagates(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	op := mustOperation(t, rt, "nil",
		func(context.Context, int) (int, error) { panic(nil) },
		Policy[int]{Effect: Pure, Resources: []Requirement{{Name: "r", Units: 1}}, Timeout: time.Hour})
	var got any
	func() {
		defer func() { got = recover() }()
		_, _ = op.Do(context.Background(), 0)
	}()
	var pn *runtime.PanicNilError
	if e, ok := got.(error); !ok || !errors.As(e, &pn) {
		t.Fatalf("recovered %v, want *runtime.PanicNilError", got)
	}
	if s := rt.Stats(); s.Admitted != 1 || s.Finished != 1 {
		t.Fatalf("stats after panic: %+v", s)
	}
	assertAccount(t, rt.budgets["r"], 0, 0)
}

func TestContractRejectedCountsOverloadAndStopping(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	rt.budgets["r"].tryAcquire(1)
	op := mustOperation(t, rt, "op",
		func(context.Context, int) (int, error) { return 0, nil },
		Policy[int]{Effect: Pure, Admission: Reject, Resources: []Requirement{{Name: "r", Units: 1}}})
	_, overload := op.Do(context.Background(), 0)
	rt.budgets["r"].release(1)
	_ = rt.Shutdown(context.Background())
	_, stopping := op.Do(context.Background(), 0)

	if !errors.Is(overload, ErrOverloaded) || stopping != ErrRuntimeStopping || OutcomeOf(stopping) != Permanent {
		t.Fatalf("errors: overload %v, stopping %v", overload, stopping)
	}
	if s := rt.Stats(); s != (Stats{Admitted: 1, Rejected: 2, Finished: 1}) {
		t.Fatalf("stats %+v, want Admitted 1, Rejected 2, Finished 1", s)
	}
}

// A caller whose context is already done takes no key and waits for no
// units: it gets its ctx.Err(), uncounted (Amendment 1, v0.2's x/sync
// Acquire). Only an unkeyed Reject call ignores ctx and reports overload.
func TestContractDoneCallerTakesNothing(t *testing.T) {
	cases := []struct {
		name      string
		mode      AdmissionMode
		keyed     bool
		keyHeld   bool
		unitsHeld bool
		want      error
		rejected  uint64
	}{
		{"reject/key-held", Reject, true, true, false, context.Canceled, 0},
		{"reject/key-free/units-held", Reject, true, false, true, context.Canceled, 0},
		{"reject/unkeyed/units-held", Reject, false, false, true, ErrOverloaded, 1},
		{"reject/unkeyed/units-free", Reject, false, false, false, context.Canceled, 0},
		{"wait/key-held", Wait, true, true, false, context.Canceled, 0},
		{"wait/unkeyed/units-held", Wait, false, false, true, context.Canceled, 0},
		{"wait/unkeyed/units-free", Wait, false, false, false, context.Canceled, 0},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
			p := Policy[int]{Effect: Pure, Admission: c.mode, Resources: []Requirement{{Name: "r", Units: 1}}}
			if c.keyed {
				p.SerializeKey = func(int) string { return "k" }
			}
			op := mustOperation(t, rt, "op", func(context.Context, int) (int, error) { return 0, nil }, p)
			if c.keyHeld {
				rt.keys.lock(context.Background(), "k", Wait)
			}
			if c.unitsHeld {
				rt.budgets["r"].tryAcquire(1)
			}
			_, err := op.Do(ctx, 0)
			if !errors.Is(err, c.want) {
				t.Fatalf("Do: %v, want %v", err, c.want)
			}
			if s := rt.Stats(); s.Rejected != c.rejected || s.Started != 0 {
				t.Fatalf("stats %+v, want Rejected %d, Started 0", s, c.rejected)
			}
			if c.keyed && !c.keyHeld && len(rt.keys.m) != 0 {
				t.Fatal("a done caller left a key entry behind")
			}
		})
	}
}

func TestContractUnknownAdmissionWaits(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	r := rt.budgets["r"]
	r.tryAcquire(1)
	op := mustOperation(t, rt, "op",
		func(context.Context, int) (int, error) { return 0, nil },
		Policy[int]{Effect: Pure, Admission: AdmissionMode(99), Resources: []Requirement{{Name: "r", Units: 1}}})
	done := make(chan error, 1)
	go func() { _, err := op.Do(context.Background(), 0); done <- err }()
	waitFor(t, func() bool { return queued(r) == 1 })
	r.release(1)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// Stats snapshots never show Started or Finished above Admitted, even
// while calls race through Do.
func TestContractStatsSnapshotsAreConsistent(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 2})
	op := mustOperation(t, rt, "op",
		func(context.Context, int) (int, error) { return 0, nil },
		Policy[int]{Effect: Pure, Resources: []Requirement{{Name: "r", Units: 1}}})
	var stop atomic.Bool
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !stop.Load() {
				_, _ = op.Do(context.Background(), 0)
			}
		}()
	}
	for range 20_000 {
		if s := rt.Stats(); s.Started > s.Admitted || s.Finished > s.Admitted {
			stop.Store(true)
			t.Fatalf("inconsistent snapshot %+v", s)
		}
	}
	stop.Store(true)
	wg.Wait()
}

func TestContractNonIdempotentRetriesOnlyTransient(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	for _, tc := range []struct {
		o    Outcome
		want int
	}{{Transient, 3}, {Uncertain, 1}, {Throttled, 1}, {Permanent, 1}} {
		calls := 0
		op := mustOperation(t, rt, "charge",
			func(context.Context, int) (int, error) { calls++; return 0, Fail(tc.o, errors.New("x")) },
			Policy[int]{Effect: NonIdempotent, Retry: RetryPolicy{MaxAttempts: 3, On: map[Outcome]bool{Transient: true}}})
		if _, err := op.Do(context.Background(), 0); OutcomeOf(err) != tc.o || calls != tc.want {
			t.Fatalf("%v: %d calls, outcome %v; want %d calls", tc.o, calls, OutcomeOf(err), tc.want)
		}
	}
}

func TestContractRendering(t *testing.T) {
	cause := errors.New("cause")
	for got, want := range map[string]string{
		Success.String():                       "Success",
		Transient.String():                     "Transient",
		Permanent.String():                     "Permanent",
		Throttled.String():                     "Throttled",
		Uncertain.String():                     "Uncertain",
		Outcome(9).String():                    "Outcome(9)",
		Idempotent.String():                    "Idempotent",
		Effect(7).String():                     "Effect(7)",
		Fail(Throttled, cause).Error():         "cause",
		(&Failure{Outcome: Uncertain}).Error(): "execution: failure(Uncertain)",
		newResource("db-write", 4).String():    "db-write(4)",
	} {
		if got != want {
			t.Errorf("rendered %q, want %q", got, want)
		}
	}
}

// A handler that returns a nil *Failure by mistake (var f *Failure;
// return v, f) hands Do a non-nil error with no verdict. It must read as
// untagged: Permanent, never retried, and safe to print, wrap and inspect.
func TestContractTypedNilFailure(t *testing.T) {
	var nilFailure *Failure
	var err error = nilFailure
	if o := OutcomeOf(err); o != Permanent {
		t.Fatalf("OutcomeOf(nil *Failure) = %v, want Permanent", o)
	}
	if o := OutcomeOf(Fail(Transient, err)); o != Transient {
		t.Fatalf("Fail(Transient, nil *Failure) reads %v, want Transient", o)
	}
	_ = err.Error()
	_ = errors.Is(err, context.Canceled)

	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	var calls atomic.Int32
	op := mustOperation(t, rt, "mistake",
		func(context.Context, int) (int, error) { calls.Add(1); return 0, nilFailure },
		Policy[int]{Effect: Idempotent, Resources: []Requirement{{Name: "r", Units: 1}},
			Retry: RetryPolicy{MaxAttempts: 3, On: map[Outcome]bool{Transient: true, Uncertain: true}}})
	_, err = op.Do(context.Background(), 0)
	if err == nil || OutcomeOf(err) != Permanent {
		t.Fatalf("Do = %v (%v), want a Permanent error", err, OutcomeOf(err))
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("handler ran %d times, want 1 (untagged is never retried)", n)
	}
	if s := rt.Stats(); s.Finished != s.Admitted || s.Retried != 0 {
		t.Fatalf("stats %+v: call not settled", s)
	}
}

// Stats reads its counters in reverse increment order, so no snapshot taken
// under load shows Started or Finished above Admitted. The window a wrong
// order opens is a few nanoseconds, so half the processors call Do on the
// fastest path while the other half take snapshots.
func TestContractStatsSnapshotOrdered(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	op := mustOperation(t, rt, "op",
		func(_ context.Context, i int) (int, error) { return i, nil },
		Policy[int]{Effect: Pure})
	half := max(runtime.GOMAXPROCS(0)/2, 1)
	var stop atomic.Bool
	var bad atomic.Pointer[Stats]
	var callers, readers sync.WaitGroup
	for range half {
		callers.Go(func() {
			for !stop.Load() {
				_, _ = op.Do(context.Background(), 0)
			}
		})
	}
	for range half {
		readers.Go(func() {
			for range 2_000_000 {
				if s := rt.Stats(); s.Started > s.Admitted || s.Finished > s.Admitted {
					bad.Store(&s)
					return
				}
			}
		})
	}
	readers.Wait()
	stop.Store(true)
	callers.Wait()
	if s := bad.Load(); s != nil {
		t.Fatalf("snapshot %+v shows more started or finished than admitted", *s)
	}
}

// When Retry.On names several outcomes that can never be retried, the
// lowest one is reported, whatever the map's iteration order.
func TestContractRetryTableReportsLowestInvalid(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	on := map[Outcome]bool{Throttled: true, Permanent: true, Success: true}
	for range 50 {
		_, err := NewOperation(rt, "op",
			func(_ context.Context, i int) (int, error) { return i, nil },
			Policy[int]{Effect: Pure, Retry: RetryPolicy{MaxAttempts: 2, On: on}})
		if !errors.Is(err, ErrInvalidRetryPolicy) || !strings.Contains(err.Error(), "retry on Success") {
			t.Fatalf("got %v, want ErrInvalidRetryPolicy naming Success", err)
		}
	}
}
