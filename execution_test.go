package execution

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// inFlight tracks current + peak concurrent handlers (the budget probe).
type inFlight struct {
	mu  sync.Mutex
	cur int
	max int
}

func (f *inFlight) enter() {
	f.mu.Lock()
	f.cur++
	if f.cur > f.max {
		f.max = f.cur
	}
	f.mu.Unlock()
}

func (f *inFlight) exit() {
	f.mu.Lock()
	f.cur--
	f.mu.Unlock()
}

func (f *inFlight) peak() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.max
}

// gate releases waiters when closed; handlers block on it.
type gate chan struct{}

func newGate() gate  { return make(chan struct{}) }
func (g gate) open() { close(g) }
func (g gate) wait() { <-g }
func (g gate) waitCtx(ctx context.Context) error {
	select {
	case <-g:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func mustRuntime(t *testing.T, specs ...ResourceSpec) *Runtime {
	t.Helper()
	rt, err := NewRuntime(specs...)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	return rt
}

func mustOperation[I any, O any](t *testing.T, rt *Runtime, name string, h Handler[I, O], p Policy[I]) *Operation[I, O] {
	t.Helper()
	op, err := NewOperation(rt, name, h, p)
	if err != nil {
		t.Fatalf("NewOperation(%s): %v", name, err)
	}
	return op
}

// ---------------------------------------------------------------------------
// construction-time validation
// ---------------------------------------------------------------------------

func TestNewRuntimeValidation(t *testing.T) {
	if _, err := NewRuntime(); err == nil {
		t.Fatal("want error for zero resources")
	}
	if _, err := NewRuntime(ResourceSpec{Name: "a", Capacity: 0}); err == nil {
		t.Fatal("want error for capacity 0")
	}
	if _, err := NewRuntime(ResourceSpec{Name: "a", Capacity: -1}); err == nil {
		t.Fatal("want error for negative capacity")
	}
	if _, err := NewRuntime(
		ResourceSpec{Name: "a", Capacity: 1},
		ResourceSpec{Name: "a", Capacity: 2}); err == nil {
		t.Fatal("want error for duplicate resource")
	}
	if _, err := NewRuntime(ResourceSpec{Name: "", Capacity: 1}); err == nil {
		t.Fatal("want error for empty name")
	}
	if _, err := NewRuntime(ResourceSpec{Name: "ok", Capacity: 1}); err != nil {
		t.Fatalf("valid runtime rejected: %v", err)
	}
}

func TestNewOperationValidation(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 2})

	cases := []struct {
		label  string
		policy Policy[int]
	}{
		{"unknown resource", Policy[int]{Resources: []Requirement{{Name: "nope", Units: 1}}}},
		{"zero units", Policy[int]{Resources: []Requirement{{Name: "r", Units: 0}}}},
		{"negative units", Policy[int]{Resources: []Requirement{{Name: "r", Units: -1}}}},
		{"units above capacity", Policy[int]{Resources: []Requirement{{Name: "r", Units: 3}}}},
		{"duplicate resource", Policy[int]{Resources: []Requirement{{Name: "r", Units: 1}, {Name: "r", Units: 1}}}},
		{"unsafe retry contract", Policy[int]{Effect: NonIdempotent, Retry: RetryPolicy{MaxAttempts: 2, On: map[Outcome]bool{Uncertain: true}}}},
	}
	for _, tc := range cases {
		if _, err := NewOperation(rt, "op", func(ctx context.Context, i int) (int, error) { return i, nil }, tc.policy); err == nil {
			t.Errorf("%s: want construction error", tc.label)
		}
	}

	if _, err := NewOperation[int, int](rt, "", func(ctx context.Context, i int) (int, error) { return i, nil }, Policy[int]{}); err == nil {
		t.Error("empty name: want construction error")
	}
	if _, err := NewOperation[int, int](rt, "op", nil, Policy[int]{}); err == nil {
		t.Error("nil handler: want construction error")
	}
	if _, err := NewOperation[int, int](nil, "op", func(ctx context.Context, i int) (int, error) { return i, nil }, Policy[int]{}); err == nil {
		t.Error("nil runtime: want construction error")
	}
}

// The PRD's headline differentiator, verbatim: NonIdempotent + retry on
// Uncertain must be structurally rejected before serving traffic.
func TestUnsafeContractRejectedAtConstruction(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "renderer", Capacity: 8})
	_, err := NewOperation(rt, "render",
		func(ctx context.Context, r string) (string, error) { return r, nil },
		Policy[string]{
			Effect:    NonIdempotent,
			Resources: []Requirement{{Name: "renderer", Units: 1}},
			Retry:     RetryPolicy{MaxAttempts: 2, On: map[Outcome]bool{Uncertain: true}},
		})
	if !errors.Is(err, ErrUnsafeRetry) {
		t.Fatalf("want ErrUnsafeRetry, got %v", err)
	}
	if !strings.Contains(err.Error(), "non-idempotent") {
		t.Fatalf("error should name the violation, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// PRD matrix
// ---------------------------------------------------------------------------

// capacity=8, 20 Render calls → max simultaneous handlers = exactly 8.
func TestCapacityLimitHeldUnderLoad(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "renderer", Capacity: 8})
	tr := &inFlight{}
	release := newGate()

	op := mustOperation(t, rt, "render",
		func(ctx context.Context, _ int) (int, error) {
			tr.enter()
			defer tr.exit()
			release.wait()
			return 1, nil
		},
		Policy[int]{
			Resources: []Requirement{{Name: "renderer", Units: 1}},
			Admission: Wait,
			Timeout:   5 * time.Second,
		})

	const calls = 20
	var wg sync.WaitGroup
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = op.Do(context.Background(), i)
		}()
	}
	time.Sleep(150 * time.Millisecond) // let admitted ops reach the gate
	if got := tr.peak(); got != 8 {
		t.Fatalf("peak concurrency = %d, want exactly 8", got)
	}
	release.open()
	wg.Wait()
	if tr.peak() != 8 {
		t.Fatalf("peak moved after release: %d", tr.peak())
	}
}

// Render + Reject, capacity exhausted → immediate ErrOverloaded (Throttled).
func TestRejectAdmissionFailsFast(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "renderer", Capacity: 1})
	release := newGate()
	holder := mustOperation(t, rt, "hold",
		func(ctx context.Context, _ int) (int, error) { release.wait(); return 0, nil },
		Policy[int]{Resources: []Requirement{{Name: "renderer", Units: 1}}, Admission: Wait})
	holderDone := make(chan struct{})
	go func() { _, _ = holder.Do(context.Background(), 0); close(holderDone) }()
	time.Sleep(50 * time.Millisecond) // holder owns capacity now

	reject := mustOperation(t, rt, "render",
		func(ctx context.Context, _ int) (int, error) { return 0, nil },
		Policy[int]{Resources: []Requirement{{Name: "renderer", Units: 1}}, Admission: Reject})

	start := time.Now()
	_, err := reject.Do(context.Background(), 0)
	elapsed := time.Since(start)

	if !errors.Is(err, ErrOverloaded) {
		t.Fatalf("want ErrOverloaded, got %v", err)
	}
	if OutcomeOf(err) != Throttled {
		t.Fatalf("want OutcomeOf=Throttled, got %v", OutcomeOf(err))
	}
	if elapsed > 50*time.Millisecond {
		t.Fatalf("Reject must fail immediately, took %v", elapsed)
	}
	release.open()
	<-holderDone
}

// Events + Catalogue share db-write: aggregate usage never exceeds capacity.
func TestSharedBudgetAcrossOperations(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "db-write", Capacity: 4})
	tr := &inFlight{}
	release := newGate()

	mk := func(name string, units int64, effect Effect) *Operation[int, int] {
		return mustOperation(t, rt, name,
			func(ctx context.Context, _ int) (int, error) {
				tr.enter()
				defer tr.exit()
				release.wait()
				return 0, nil
			},
			Policy[int]{
				Effect:    effect,
				Resources: []Requirement{{Name: "db-write", Units: units}},
				Admission: Wait,
			})
	}
	events := mk("events.record", 1, Idempotent)
	cat := mk("catalogue.replace", 3, Idempotent)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); _, _ = events.Do(context.Background(), i) }(i)
		wg.Add(1)
		go func(i int) { defer wg.Done(); _, _ = cat.Do(context.Background(), i) }(i)
	}
	time.Sleep(250 * time.Millisecond)
	if got := tr.peak(); got > 4 {
		t.Fatalf("aggregate peak = %d, budget is 4", got)
	}
	release.open()
	wg.Wait()
}

// Catalogue(A) vs Catalogue(A): never overlap. A vs B: overlap when the
// budget permits (both in flight together).
func TestKeySerializationSameKeySerialDifferentKeyParallel(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "db-write", Capacity: 8})
	release := newGate()
	tr := &inFlight{}
	seen := map[string]int{}
	var seenMu sync.Mutex

	op := mustOperation(t, rt, "catalogue.replace",
		func(ctx context.Context, brand string) (int, error) {
			seenMu.Lock()
			seen[brand]++
			seenMu.Unlock()
			tr.enter()
			defer tr.exit()
			release.wait()
			return 0, nil
		},
		Policy[string]{
			Effect:       Idempotent,
			Resources:    []Requirement{{Name: "db-write", Units: 1}},
			Admission:    Wait,
			SerializeKey: func(b string) string { return b },
		})

	start := make(chan struct{})
	var wg sync.WaitGroup
	run := func(brand string) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _ = op.Do(context.Background(), brand)
		}()
	}
	for i := 0; i < 4; i++ {
		run("A")
		run("B")
	}
	close(start)
	time.Sleep(250 * time.Millisecond)

	// Both brands hold the (ample) budget simultaneously → cross-key
	// parallelism; same key never overlaps.
	if got := tr.peak(); got != 2 {
		t.Fatalf("peak in-handler = %d, want exactly 2 (A and B concurrent)", got)
	}
	release.open()
	wg.Wait()

	seenMu.Lock()
	defer seenMu.Unlock()
	for brand, n := range seen {
		if n != 4 {
			t.Fatalf("brand %s ran %d times, want 4", brand, n)
		}
	}
}

// Waiting request context cancelled → leaves immediately, consumes no
// capacity afterwards.
func TestWaitingCallerCancellationLeavesClean(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "renderer", Capacity: 1})
	release := newGate()
	holder := mustOperation(t, rt, "hold",
		func(ctx context.Context, _ int) (int, error) { release.waitCtx(ctx); return 0, ctx.Err() },
		Policy[int]{Resources: []Requirement{{Name: "renderer", Units: 1}}, Admission: Wait})
	holderDone := make(chan struct{})
	go func() { _, _ = holder.Do(context.Background(), 0); close(holderDone) }()
	time.Sleep(50 * time.Millisecond)

	waiter := mustOperation(t, rt, "waiter",
		func(ctx context.Context, _ int) (int, error) { return 0, nil },
		Policy[int]{Resources: []Requirement{{Name: "renderer", Units: 1}}, Admission: Wait})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := waiter.Do(ctx, 0); done <- err }()
	time.Sleep(50 * time.Millisecond) // waiter parked on the semaphore
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled waiter did not leave promptly")
	}

	// Drain the holder, then prove zero leakage: the cancelled waiter must
	// have consumed nothing, so the full capacity is free for a Reject probe.
	release.open()
	<-holderDone
	probe := mustOperation(t, rt, "probe",
		func(ctx context.Context, _ int) (int, error) { return 0, nil },
		Policy[int]{Resources: []Requirement{{Name: "renderer", Units: 1}}, Admission: Reject})
	if _, err := probe.Do(context.Background(), 0); err != nil {
		t.Fatalf("capacity leaked after cancellation: %v", err)
	}
}

// Handler panic → key and resources released, then the panic propagates.
func TestPanicReleasesEverythingAndPropagates(t *testing.T) {
	rt := mustRuntime(t,
		ResourceSpec{Name: "renderer", Capacity: 1},
		ResourceSpec{Name: "db-write", Capacity: 1})

	op := mustOperation(t, rt, "boom",
		func(ctx context.Context, _ string) (int, error) { panic("programmer bug") },
		Policy[string]{
			Resources:    []Requirement{{Name: "renderer", Units: 1}, {Name: "db-write", Units: 1}},
			Admission:    Wait,
			SerializeKey: func(string) string { return "brand-1" },
		})

	recovered := make(chan any, 1)
	go func() {
		defer func() { recovered <- recover() }()
		_, _ = op.Do(context.Background(), "x")
	}()
	if r := <-recovered; r != "programmer bug" {
		t.Fatalf("panic must propagate, got %v", r)
	}

	// Both resources are back: a Reject probe takes one unit of each.
	probe := mustOperation(t, rt, "probe",
		func(ctx context.Context, _ string) (int, error) { return 0, nil },
		Policy[string]{
			Resources: []Requirement{{Name: "renderer", Units: 1}, {Name: "db-write", Units: 1}},
			Admission: Reject,
		})
	if _, err := probe.Do(context.Background(), ""); err != nil {
		t.Fatalf("resources leaked after panic: %v", err)
	}

	// The key was released: another op acquires the same key.
	withKey := mustOperation(t, rt, "withkey",
		func(ctx context.Context, _ string) (int, error) { return 0, nil },
		Policy[string]{SerializeKey: func(s string) string { return "brand-1" }})
	if _, err := withKey.Do(context.Background(), ""); err != nil {
		t.Fatalf("key leaked after panic: %v", err)
	}
}

// Shutdown → new executions rejected, active ones drain to completion.
func TestShutdownRejectsNewAndDrainsActive(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "db-write", Capacity: 3})
	release := newGate()
	var mu sync.Mutex
	drained := 0

	op := mustOperation(t, rt, "work",
		func(ctx context.Context, _ int) (int, error) {
			release.wait()
			mu.Lock()
			drained++
			mu.Unlock()
			return 0, nil
		},
		Policy[int]{Resources: []Requirement{{Name: "db-write", Units: 1}}, Admission: Wait})

	const active = 3
	var wg sync.WaitGroup
	for i := 0; i < active; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = op.Do(context.Background(), 0) }()
	}
	time.Sleep(100 * time.Millisecond) // all three executing

	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- rt.Shutdown(context.Background()) }()
	time.Sleep(50 * time.Millisecond) // shutdown begun

	// New work is rejected...
	if _, err := op.Do(context.Background(), 0); !errors.Is(err, ErrRuntimeStopping) {
		t.Fatalf("want ErrRuntimeStopping after shutdown, got %v", err)
	}

	// ...and active work drains.
	release.open()
	wg.Wait()
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatalf("Shutdown: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Shutdown did not return after drain")
	}
	mu.Lock()
	defer mu.Unlock()
	if drained != active {
		t.Fatalf("drained %d, want %d", drained, active)
	}
}

// ---------------------------------------------------------------------------
// added invariants
// ---------------------------------------------------------------------------

// Concurrent Do vs Shutdown race hammer. Invariant: Finished == Admitted
// after the dust settles — every admitted operation drains exactly once.
func TestRaceHammerDoVsShutdown(t *testing.T) {
	for iter := 0; iter < 50; iter++ {
		rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 4})
		op := mustOperation(t, rt, "spin",
			func(ctx context.Context, _ int) (int, error) { return iter, nil },
			Policy[int]{Resources: []Requirement{{Name: "r", Units: 1}}, Admission: Wait})

		var wg sync.WaitGroup
		for g := 0; g < 8; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 25; i++ {
					_, _ = op.Do(context.Background(), i) // may legally fail with stopping
				}
			}()
		}
		time.Sleep(time.Duration(iter%7) * time.Millisecond)
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

// Partial multi-resource acquire must roll everything back: an op needing
// {a,b} under Reject admission, where a is free but b is held, fails
// WITHOUT keeping 'a'.
func TestPartialAcquireRollsBack(t *testing.T) {
	rt := mustRuntime(t,
		ResourceSpec{Name: "a", Capacity: 1},
		ResourceSpec{Name: "b", Capacity: 1})
	release := newGate()

	holder := mustOperation(t, rt, "hold-b",
		func(ctx context.Context, _ int) (int, error) { release.wait(); return 0, nil },
		Policy[int]{Resources: []Requirement{{Name: "b", Units: 1}}, Admission: Wait})
	holderDone := make(chan struct{})
	go func() { _, _ = holder.Do(context.Background(), 0); close(holderDone) }()
	time.Sleep(50 * time.Millisecond) // holder owns b

	victim := mustOperation(t, rt, "needs-a-and-b",
		func(ctx context.Context, _ int) (int, error) { return 0, nil },
		Policy[int]{Resources: []Requirement{{Name: "a", Units: 1}, {Name: "b", Units: 1}}, Admission: Reject})
	_, err := victim.Do(context.Background(), 0)
	if !errors.Is(err, ErrOverloaded) {
		t.Fatalf("want ErrOverloaded, got %v", err)
	}

	// Rollback proof: 'a' must be free right now (Reject probe succeeds).
	probeA := mustOperation(t, rt, "probe-a",
		func(ctx context.Context, _ int) (int, error) { return 0, nil },
		Policy[int]{Resources: []Requirement{{Name: "a", Units: 1}}, Admission: Reject})
	if _, err := probeA.Do(context.Background(), 0); err != nil {
		t.Fatalf("'a' leaked after partial-acquire failure: %v", err)
	}
	// And 'b' is still held by the holder (Reject probe fails).
	probeB := mustOperation(t, rt, "probe-b",
		func(ctx context.Context, _ int) (int, error) { return 0, nil },
		Policy[int]{Resources: []Requirement{{Name: "b", Units: 1}}, Admission: Reject})
	if _, err := probeB.Do(context.Background(), 0); !errors.Is(err, ErrOverloaded) {
		t.Fatalf("'b' should still be held, got %v", err)
	}

	release.open()
	<-holderDone
}

// Deterministic global resource ordering: an op blocked on the FIRST
// resource in global order holds nothing, so an op needing only a LATER
// resource proceeds. (Policy lists [b, a] to prove normalization.)
func TestGlobalOrderingPreventsHoldAndWaitStarvation(t *testing.T) {
	rt := mustRuntime(t,
		ResourceSpec{Name: "a", Capacity: 1},
		ResourceSpec{Name: "b", Capacity: 1})
	release := newGate()

	aHolder := mustOperation(t, rt, "hold-a",
		func(ctx context.Context, _ int) (int, error) { release.wait(); return 0, nil },
		Policy[int]{Resources: []Requirement{{Name: "a", Units: 1}}, Admission: Wait})
	aHolderDone := make(chan struct{})
	go func() { _, _ = aHolder.Do(context.Background(), 0); close(aHolderDone) }()
	time.Sleep(50 * time.Millisecond)

	victim := mustOperation(t, rt, "needs-b-then-a",
		func(ctx context.Context, _ int) (int, error) { return 0, nil },
		Policy[int]{Resources: []Requirement{{Name: "b", Units: 1}, {Name: "a", Units: 1}}, Admission: Wait})
	victimDone := make(chan struct{})
	go func() { _, _ = victim.Do(context.Background(), 0); close(victimDone) }()
	time.Sleep(50 * time.Millisecond) // victim blocked on 'a' (global first)

	// If ordering followed policy order, the victim would hold 'b' while
	// waiting on 'a' — and this b-only op would block. Global order ⇒ free.
	bOnly := mustOperation(t, rt, "needs-b",
		func(ctx context.Context, _ int) (int, error) { return 0, nil },
		Policy[int]{Resources: []Requirement{{Name: "b", Units: 1}}, Admission: Reject})
	if _, err := bOnly.Do(context.Background(), 0); err != nil {
		t.Fatalf("victim holds 'b' while waiting — global ordering broken: %v", err)
	}

	release.open()
	<-aHolderDone
	<-victimDone
}

// Unclassified error ⇒ Permanent ⇒ never retried.
func TestUnclassifiedErrorIsPermanentAndNotRetried(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	calls := 0
	op := mustOperation(t, rt, "op",
		func(ctx context.Context, _ int) (int, error) {
			calls++
			return 0, errors.New("some plain error")
		},
		Policy[int]{
			Resources: []Requirement{{Name: "r", Units: 1}},
			Retry:     RetryPolicy{MaxAttempts: 5, On: map[Outcome]bool{Transient: true}},
		})
	_, err := op.Do(context.Background(), 0)
	if err == nil || err.Error() != "some plain error" {
		t.Fatalf("plain error must pass through, got %v", err)
	}
	if OutcomeOf(err) != Permanent {
		t.Fatalf("unclassified must be Permanent, got %v", OutcomeOf(err))
	}
	if calls != 1 {
		t.Fatalf("Permanent must not retry, calls=%d", calls)
	}
}

// Fresh cooperative timeout per attempt: attempt 2 starts with a full
// budget, and an unclassified deadline hit promotes to Uncertain.
func TestFreshTimeoutPerAttempt(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	const timeout = 60 * time.Millisecond

	var startOffsets []time.Duration
	base := time.Now()
	op := mustOperation(t, rt, "op",
		func(ctx context.Context, _ int) (int, error) {
			startOffsets = append(startOffsets, time.Since(base))
			select {
			case <-time.After(10 * time.Second):
				return 0, nil
			case <-ctx.Done():
				return 0, ctx.Err() // unclassified DeadlineExceeded
			}
		},
		Policy[int]{
			Effect:    Idempotent, // promotion to Uncertain is retryable here
			Resources: []Requirement{{Name: "r", Units: 1}},
			Timeout:   timeout,
			Retry:     RetryPolicy{MaxAttempts: 2, On: map[Outcome]bool{Uncertain: true}},
		})

	start := time.Now()
	_, err := op.Do(context.Background(), 0)
	wall := time.Since(start)

	if err == nil {
		t.Fatal("both attempts timed out; want final error")
	}
	if OutcomeOf(err) != Uncertain {
		t.Fatalf("deadline-expired unclassified should promote to Uncertain, got %v", OutcomeOf(err))
	}
	if len(startOffsets) != 2 {
		t.Fatalf("want exactly 2 attempts, got %d", len(startOffsets))
	}
	// Attempt 2 starts ~timeout after attempt 1 (fresh budget), not ~2×.
	if startOffsets[1] > 2*timeout {
		t.Fatalf("attempt 2 started at %v — budget looks shared, not fresh", startOffsets[1])
	}
	if wall > 4*timeout {
		t.Fatalf("wall time %v exceeds MaxAttempts×Timeout contract", wall)
	}
}

// Caller cancellation aborts the retry loop immediately, regardless of
// retry policy: the cancel lands DURING the (only) in-flight attempt's
// 60 ms handler wait, so attempt 2 must never start.
func TestCallerCancellationAbortsRetryLoop(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	calls := 0
	op := mustOperation(t, rt, "op",
		func(ctx context.Context, _ int) (int, error) {
			calls++
			select {
			case <-time.After(60 * time.Millisecond): // attempt takes a while
				return 0, Fail(Transient, errors.New("flaky"))
			case <-ctx.Done():
				return 0, Fail(Transient, ctx.Err())
			}
		},
		Policy[int]{
			Resources: []Requirement{{Name: "r", Units: 1}},
			Retry:     RetryPolicy{MaxAttempts: 10, On: map[Outcome]bool{Transient: true}},
		})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	_, err := op.Do(ctx, 0)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled to abort loop, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("retry loop must abort on caller cancel, calls=%d", calls)
	}
}

// Runtime shutdown wakes admission waiters — both resource waiters and
// key waiters — with ErrRuntimeStopping, while the holder drains.
func TestShutdownWakesAdmissionWaiters(t *testing.T) {
	t.Run("resource waiter", func(t *testing.T) {
		rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
		release := newGate()
		holder := mustOperation(t, rt, "hold",
			func(ctx context.Context, _ int) (int, error) { release.waitCtx(ctx); return 0, ctx.Err() },
			Policy[int]{Resources: []Requirement{{Name: "r", Units: 1}}, Admission: Wait})
		holderDone := make(chan struct{})
		go func() { _, _ = holder.Do(context.Background(), 0); close(holderDone) }()
		time.Sleep(50 * time.Millisecond)

		waiter := mustOperation(t, rt, "waiter",
			func(ctx context.Context, _ int) (int, error) { return 0, nil },
			Policy[int]{Resources: []Requirement{{Name: "r", Units: 1}}, Admission: Wait})
		errCh := make(chan error, 1)
		go func() { _, err := waiter.Do(context.Background(), 0); errCh <- err }()
		time.Sleep(50 * time.Millisecond)

		go func() { _ = rt.Shutdown(context.Background()) }()
		select {
		case err := <-errCh:
			if !errors.Is(err, ErrRuntimeStopping) {
				t.Fatalf("want ErrRuntimeStopping, got %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("resource waiter not woken by shutdown")
		}
		release.open()
		<-holderDone
	})

	t.Run("key waiter", func(t *testing.T) {
		rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
		release := newGate()
		holder := mustOperation(t, rt, "hold",
			func(ctx context.Context, _ string) (int, error) { release.waitCtx(ctx); return 0, ctx.Err() },
			Policy[string]{SerializeKey: func(s string) string { return s }})
		holderDone := make(chan struct{})
		go func() { _, _ = holder.Do(context.Background(), "A"); close(holderDone) }()
		time.Sleep(50 * time.Millisecond)

		waiter := mustOperation(t, rt, "waiter",
			func(ctx context.Context, _ string) (int, error) { return 0, nil },
			Policy[string]{SerializeKey: func(s string) string { return s }})
		errCh := make(chan error, 1)
		go func() { _, err := waiter.Do(context.Background(), "A"); errCh <- err }()
		time.Sleep(50 * time.Millisecond)

		go func() { _ = rt.Shutdown(context.Background()) }()
		select {
		case err := <-errCh:
			if !errors.Is(err, ErrRuntimeStopping) {
				t.Fatalf("want ErrRuntimeStopping, got %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("key waiter not woken by shutdown")
		}
		release.open()
		<-holderDone
	})
}

// Same-key waiter holds ZERO resource capacity while queued: if it grabbed
// resources before the key, the last unit would be stuck behind it.
func TestSameKeyWaiterHoldsNoResources(t *testing.T) {
	// db-write capacity 3. A (key "K", 1 unit) holds 1 and blocks.
	// A' (same key "K", 2 units) queues on the KEY. Under resources-first
	// ordering A' would have seized the remaining 2 units and B (1 unit)
	// could never run. Key-first ⇒ B completes while A' still waits.
	rt := mustRuntime(t, ResourceSpec{Name: "db-write", Capacity: 3})
	release := newGate()
	var mu sync.Mutex
	bDone := false

	op := mustOperation(t, rt, "catalogue.replace",
		func(ctx context.Context, brand string) (int, error) {
			release.wait()
			return 0, nil
		},
		Policy[string]{
			Effect:       Idempotent,
			Resources:    []Requirement{{Name: "db-write", Units: 1}},
			Admission:    Wait,
			SerializeKey: func(b string) string { return b },
		})

	go func() { _, _ = op.Do(context.Background(), "A") }()
	time.Sleep(80 * time.Millisecond) // A holds 1 unit, blocked on gate
	go func() { _, _ = op.Do(context.Background(), "A-prime") }()
	time.Sleep(80 * time.Millisecond) // A-prime waiting on key "A-prime"...

	// NOTE: key derives from brand, so give A' the same brand path:
	// (the go-routine above used brand "A-prime" — same-key setup is
	// covered by the identical-key holder below; this test's assertion is
	// about B proceeding while a same-key waiter exists.)

	// A same-key waiter (brand "A") queued behind the holder:
	go func() { _, _ = op.Do(context.Background(), "A") }()
	time.Sleep(80 * time.Millisecond) // now truly parked on key "A"

	bOnly := mustOperation(t, rt, "b",
		func(ctx context.Context, _ string) (int, error) {
			mu.Lock()
			bDone = true
			mu.Unlock()
			return 0, nil
		},
		Policy[string]{
			Effect:    Idempotent,
			Resources: []Requirement{{Name: "db-write", Units: 1}},
			Admission: Wait,
		})
	bErr := make(chan error, 1)
	go func() { _, err := bOnly.Do(context.Background(), "B"); bErr <- err }()
	select {
	case err := <-bErr:
		if err != nil {
			t.Fatalf("B must run while same-key waiter queues: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("B blocked — a waiter is hoarding capacity")
	}

	mu.Lock()
	defer mu.Unlock()
	if bDone != true {
		t.Fatal("B handler did not run")
	}
	release.open()
}

// MaxAttempts normalization: 0 and 1 both mean exactly one execution.
func TestMaxAttemptsNormalization(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	for _, ma := range []int{0, 1} {
		calls := 0
		op := mustOperation(t, rt, "op",
			func(ctx context.Context, _ int) (int, error) {
				calls++
				return 0, Fail(Transient, errors.New("always transient"))
			},
			Policy[int]{
				Resources: []Requirement{{Name: "r", Units: 1}},
				Retry:     RetryPolicy{MaxAttempts: ma, On: map[Outcome]bool{Transient: true}},
			})
		_, _ = op.Do(context.Background(), 0)
		if calls != 1 {
			t.Fatalf("MaxAttempts=%d must mean 1 execution, got %d", ma, calls)
		}
	}
}

// Empty SerializeKey result ⇒ no serialization: 4 calls overlap fully.
func TestEmptyKeyMeansNoSerialization(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 8})
	release := newGate()
	tr := &inFlight{}
	op := mustOperation(t, rt, "op",
		func(ctx context.Context, _ string) (int, error) {
			tr.enter()
			defer tr.exit()
			release.wait()
			return 0, nil
		},
		Policy[string]{SerializeKey: func(string) string { return "" }})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = op.Do(context.Background(), "x") }()
	}
	time.Sleep(100 * time.Millisecond)
	if tr.peak() != 4 {
		t.Fatalf("empty key must not serialize, peak=%d", tr.peak())
	}
	release.open()
	wg.Wait()
}

// Fail/OutcomeOf/Failure.Unwrap contract.
func TestFailureContract(t *testing.T) {
	if Fail(Transient, nil) != nil {
		t.Fatal("Fail(_, nil) must be nil")
	}
	inner := errors.New("boom")
	f := Fail(Transient, inner)
	if !errors.Is(f, inner) {
		t.Fatal("Failure.Unwrap must expose inner error")
	}
	if OutcomeOf(f) != Transient || OutcomeOf(inner) != Permanent || OutcomeOf(nil) != Success {
		t.Fatal("OutcomeOf defaulting broken")
	}
	// Re-wrapping keeps the innermost verdict.
	again := Fail(Permanent, f)
	if OutcomeOf(again) != Transient {
		t.Fatal("innermost classification must win")
	}
}

// Stats counters reflect reality.
func TestStatsCounters(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	op := mustOperation(t, rt, "op",
		func(ctx context.Context, _ int) (int, error) {
			return 0, Fail(Transient, errors.New("x"))
		},
		Policy[int]{
			Resources: []Requirement{{Name: "r", Units: 1}},
			Retry:     RetryPolicy{MaxAttempts: 2, On: map[Outcome]bool{Transient: true}},
		})
	_, _ = op.Do(context.Background(), 0)
	_, _ = op.Do(context.Background(), 0)

	s := rt.Stats()
	if s.Admitted != 2 || s.Started != 2 || s.Finished != 2 {
		t.Fatalf("stats = %+v, want Admitted/Started/Finished = 2", s)
	}
	if s.Retried != 2 { // one retry per Do
		t.Fatalf("Retried = %d, want 2", s.Retried)
	}
	if s.Rejected != 0 {
		t.Fatalf("Rejected = %d, want 0", s.Rejected)
	}
}
