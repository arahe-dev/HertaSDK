package hertasdk

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// One test per CONTRACT §8 delta, each proving hertasdk's behavior where
// v0.2.0 is defective.

// Δ1: Reject with a held key fails fast instead of blocking.
func TestDelta1RejectFailsFastOnHeldKey(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 4})
	release := newGate()
	keyed := func(mode AdmissionMode) *Operation[string, int] {
		return mustOperation(t, rt, "keyed",
			func(context.Context, string) (int, error) { release.wait(); return 0, nil },
			Policy[string]{Effect: Pure, Admission: mode, Resources: []Requirement{{Name: "r", Units: 1}},
				SerializeKey: func(s string) string { return s }})
	}
	holderDone := make(chan struct{})
	go func() { _, _ = keyed(Wait).Do(context.Background(), "K"); close(holderDone) }()
	waitFor(t, func() bool { return rt.Stats().Started == 1 })

	start := time.Now()
	_, err := keyed(Reject).Do(context.Background(), "K")
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("Reject on a held key took %v", elapsed)
	}
	if !errors.Is(err, ErrOverloaded) || OutcomeOf(err) != Throttled {
		t.Fatalf("want Fail(Throttled, ErrOverloaded), got %v (%v)", err, OutcomeOf(err))
	}
	if s := rt.Stats(); s.Rejected != 1 || s.Started != 1 {
		t.Fatalf("stats %+v: want the refusal counted as Rejected, nothing started", s)
	}
	release.open()
	<-holderDone
}

// Δ2: Shutdown()==nil implies Finished == Admitted at that instant.
func TestDelta2FinishedVisibleWhenShutdownReturns(t *testing.T) {
	for iter := range 1000 {
		rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 2})
		op := mustOperation(t, rt, "spin",
			func(_ context.Context, i int) (int, error) { return i, nil },
			Policy[int]{Effect: Pure, Resources: []Requirement{{Name: "r", Units: 1}}})
		var wg sync.WaitGroup
		for range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range 20 {
					_, _ = op.Do(context.Background(), i)
				}
			}()
		}
		waitFor(t, func() bool { return rt.Stats().Admitted > 0 }) // drain real work
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Fatal(err)
		}
		if s := rt.Stats(); s.Finished != s.Admitted {
			t.Fatalf("iter %d: Shutdown returned nil with Finished %d != Admitted %d", iter, s.Finished, s.Admitted)
		}
		wg.Wait()
	}
}

// deadlineIgnorer waits for its context, then fails with a plain error, as
// a typical I/O adapter does ("i/o timeout").
func deadlineIgnorer(calls *atomic.Int32, plain error) Handler[int, int] {
	return func(ctx context.Context, _ int) (int, error) {
		calls.Add(1)
		<-ctx.Done()
		return 0, plain
	}
}

// Δ3: one classification drives both the retry and the returned error, so
// a plain error after the attempt deadline is Uncertain and is retried.
func TestDelta3PlainErrorAfterAttemptDeadlineIsRetried(t *testing.T) {
	plain := errors.New("i/o timeout")
	for _, tc := range []struct {
		on        map[Outcome]bool
		wantCalls int32
	}{
		{map[Outcome]bool{Uncertain: true}, 3},
		{nil, 1},
	} {
		rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
		var calls atomic.Int32
		op := mustOperation(t, rt, "read", deadlineIgnorer(&calls, plain),
			Policy[int]{Effect: Idempotent, Timeout: 5 * time.Millisecond,
				Retry: RetryPolicy{MaxAttempts: 3, On: tc.on}})
		_, err := op.Do(context.Background(), 0)
		if OutcomeOf(err) != Uncertain || !errors.Is(err, plain) {
			t.Fatalf("On=%v: want Fail(Uncertain, plain), got %v (%v)", tc.on, err, OutcomeOf(err))
		}
		if calls.Load() != tc.wantCalls || rt.Stats().Retried != uint64(tc.wantCalls-1) {
			t.Fatalf("On=%v: calls %d, stats %+v; want %d calls", tc.on, calls.Load(), rt.Stats(), tc.wantCalls)
		}
	}
}

// Δ4: the caller's own deadline is Permanent and never retried, whatever
// Timeout and On say.
func TestDelta4CallerDeadlineIsPermanent(t *testing.T) {
	for _, tc := range []struct {
		label   string
		timeout time.Duration
		on      map[Outcome]bool
	}{
		{"no timeout", 0, nil},
		{"timeout, no retry", time.Second, nil},
		{"timeout, retry uncertain", time.Second, map[Outcome]bool{Uncertain: true}},
	} {
		rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
		var calls atomic.Int32
		op := mustOperation(t, rt, "op",
			func(ctx context.Context, _ int) (int, error) { calls.Add(1); <-ctx.Done(); return 0, ctx.Err() },
			Policy[int]{Effect: Idempotent, Timeout: tc.timeout, Retry: RetryPolicy{MaxAttempts: 3, On: tc.on}})
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		_, err := op.Do(ctx, 0)
		cancel()
		var f *Failure
		if !errors.Is(err, context.DeadlineExceeded) || errors.As(err, &f) || OutcomeOf(err) != Permanent {
			t.Fatalf("%s: want untagged DeadlineExceeded, got %v (%v)", tc.label, err, OutcomeOf(err))
		}
		if calls.Load() != 1 {
			t.Fatalf("%s: caller-ended attempt retried: %d calls", tc.label, calls.Load())
		}
	}
}

// Δ5: Retry.On is snapshotted; mutating it later, even concurrently, has
// no effect and no data race.
func TestDelta5RetryTableIsSnapshotted(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	on := map[Outcome]bool{}
	var calls atomic.Int32
	op := mustOperation(t, rt, "op",
		func(context.Context, int) (int, error) { calls.Add(1); return 0, Fail(Transient, errors.New("x")) },
		Policy[int]{Effect: Pure, Retry: RetryPolicy{MaxAttempts: 5, On: on}})

	stop := make(chan struct{})
	mutated := make(chan struct{})
	go func() {
		defer close(mutated)
		for {
			select {
			case <-stop:
				return
			default:
				on[Transient] = true
			}
		}
	}()
	for range 100 {
		_, _ = op.Do(context.Background(), 0)
	}
	close(stop)
	<-mutated
	if calls.Load() != 100 {
		t.Fatalf("post-construction On mutation took effect: %d calls for 100 Do", calls.Load())
	}
}

// Δ6: a waiter whose caller cancelled reports the caller's error even when
// Shutdown wakes it.
func TestDelta6CallerErrorBeatsShutdown(t *testing.T) {
	for iter := range 200 {
		rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
		r := rt.budgets["r"]
		r.tryAcquire(1)
		op := mustOperation(t, rt, "w",
			func(context.Context, int) (int, error) { return 0, nil },
			Policy[int]{Effect: Pure, Resources: []Requirement{{Name: "r", Units: 1}}})
		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() { _, err := op.Do(ctx, 0); errCh <- err }()
		waitFor(t, func() bool { return queued(r) == 1 })
		cancel()
		go func() { _ = rt.Shutdown(context.Background()) }()
		if err := <-errCh; !errors.Is(err, context.Canceled) {
			t.Fatalf("iter %d: want context.Canceled, got %v", iter, err)
		}
	}
}

// Δ7: the attempt context is cancelled when the handler panics.
func TestDelta7PanicCancelsAttemptContext(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	var seen context.Context
	op := mustOperation(t, rt, "boom",
		func(ctx context.Context, _ int) (int, error) { seen = ctx; panic("bug") },
		Policy[int]{Effect: Pure, Timeout: time.Hour})
	func() {
		defer func() { _ = recover() }()
		_, _ = op.Do(context.Background(), 0)
	}()
	if seen == nil || !errors.Is(seen.Err(), context.Canceled) {
		t.Fatalf("attempt context after panic: %v, want Canceled", seen.Err())
	}
}

// Δ8: repeated timed-out Shutdown calls start no goroutines.
func TestDelta8TimedOutShutdownParksNothing(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	release := newGate()
	op := mustOperation(t, rt, "stuck",
		func(context.Context, int) (int, error) { release.wait(); return 0, nil },
		Policy[int]{Effect: Pure})
	done := make(chan struct{})
	go func() { _, _ = op.Do(context.Background(), 0); close(done) }()
	waitFor(t, func() bool { return rt.Stats().Started == 1 })

	base := runtime.NumGoroutine()
	for range 50 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		if err := rt.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("want DeadlineExceeded while draining, got %v", err)
		}
		cancel()
	}
	if got := runtime.NumGoroutine(); got > base {
		t.Fatalf("goroutines %d -> %d after 50 timed-out Shutdowns", base, got)
	}
	release.open()
	<-done
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// Δ9: Started counts only calls whose handler ran.
func TestDelta9StartedMeansHandlerRan(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	op := mustOperation(t, rt, "op",
		func(context.Context, int) (int, error) { t.Error("handler ran on a cancelled call"); return 0, nil },
		Policy[int]{Effect: Pure})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := op.Do(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if s := rt.Stats(); s.Started != 0 || s.Admitted != 1 || s.Finished != 1 {
		t.Fatalf("stats %+v: want Admitted=Finished=1, Started=0", s)
	}
}

// Δ10: an uncontended call runs its handler without any helper goroutine
// (and, per budget_test.go, without allocating).
func TestDelta10NoGoroutinePerDo(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	var during int
	op := mustOperation(t, rt, "op",
		func(context.Context, int) (int, error) { during = runtime.NumGoroutine(); return 0, nil },
		Policy[int]{Effect: Pure, Resources: []Requirement{{Name: "r", Units: 1}},
			SerializeKey: func(int) string { return "k" }})
	before := runtime.NumGoroutine()
	if _, err := op.Do(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if during != before {
		t.Fatalf("goroutines during Do: %d, before: %d", during, before)
	}
}
