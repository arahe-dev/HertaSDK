package execution

import (
	"context"
	"errors"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

// The performance budgets of CONTRACT §10, asserted as tests. Allocation,
// byte and goroutine budgets are exact. sync.Pool drops items at random
// under -race, so the pooled key entry is checked only without it. The
// latency budget is checked only outside -race and -short.

// raceEnabled is set by race_test.go when built with -race.
var raceEnabled bool

func allocsPerDo(t *testing.T, op *Operation[int, int]) float64 {
	t.Helper()
	ctx := context.Background()
	return testing.AllocsPerRun(2000, func() {
		if _, err := op.Do(ctx, 1); err != nil {
			t.Fatal(err)
		}
	})
}

func TestBudgetUncontendedAllocatesNothing(t *testing.T) {
	if got := allocsPerDo(t, uncontendedOp(t, 0, nil)); got != 0 {
		t.Fatalf("uncontended Do: %v allocs, budget 0", got)
	}
}

func TestBudgetTimeoutCostsOnlyItsContext(t *testing.T) {
	bare := testing.AllocsPerRun(2000, func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
		_ = ctx.Err()
		cancel()
	})
	if got := allocsPerDo(t, uncontendedOp(t, time.Hour, nil)); got > bare {
		t.Fatalf("Do with Timeout: %v allocs, budget %v (a bare WithTimeout)", got, bare)
	}
}

func TestBudgetKeyedAllocations(t *testing.T) {
	if raceEnabled {
		t.Skip("sync.Pool drops items under -race")
	}
	got := allocsPerDo(t, uncontendedOp(t, 0, func(int) string { return "brand-1" }))
	if got != 0 {
		t.Fatalf("keyed Do: %v allocs, want 0 amortized (key entries are pooled)", got)
	}
	t.Logf("keyed Do: %v allocs amortized", got)
}

func TestBudgetRetryAddsNoAllocations(t *testing.T) {
	flaky := Fail(Transient, errors.New("flaky")) // tagged once, reused
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	op := mustOperation(t, rt, "flaky",
		func(context.Context, int) (int, error) { return 0, flaky },
		Policy[int]{Effect: Idempotent, Resources: []Requirement{{Name: "r", Units: 1}},
			Retry: RetryPolicy{MaxAttempts: 3, On: map[Outcome]bool{Transient: true}}})
	allocs := testing.AllocsPerRun(2000, func() { _, _ = op.Do(context.Background(), 0) })
	if allocs != 0 {
		t.Fatalf("3 attempts: %v allocs, budget 0", allocs)
	}
	if s := rt.Stats(); s.Retried != 2*s.Started {
		t.Fatalf("expected 2 retries per call, stats %+v", s)
	}
}

// TestBudgetBlockedWaiter measures a Do that must queue: ≤ 2 allocations
// and ≤ 256 B per blocked call, excluding the caller's goroutine stack.
// The same hand-off loop is measured with an uncontended Do as a baseline,
// so the harness's own channel traffic is not charged to the waiter. Parking
// goroutines also makes the scheduler allocate now and then (new Ms,
// sudogs): a few per thousand calls, allowed by the slack below. A third
// allocation per call would read as 3.0, so the slack hides no regression.
func TestBudgetBlockedWaiter(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	op := mustOperation(t, rt, "w",
		func(_ context.Context, i int) (int, error) { return i, nil },
		Policy[int]{Effect: Pure, Resources: []Requirement{{Name: "r", Units: 1}}})
	r := rt.budgets["r"]
	start, done := make(chan struct{}), make(chan error)
	go func() {
		for range start {
			_, err := op.Do(context.Background(), 1)
			done <- err
		}
	}()
	defer close(start)

	call := func(blocked bool) {
		if blocked {
			r.tryAcquire(1)
		}
		start <- struct{}{}
		if blocked {
			for queued(r) != 1 { // not waitFor: t.Helper allocates
				runtime.Gosched()
			}
			r.release(1)
		}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	measure := func(blocked bool) (allocs, bytes float64) {
		const n = 2000
		for range n { // warm the sudog caches and the scheduler's Ms
			call(blocked)
		}
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		for range n {
			call(blocked)
		}
		runtime.ReadMemStats(&after)
		return float64(after.Mallocs-before.Mallocs) / n, float64(after.TotalAlloc-before.TotalAlloc) / n
	}
	baseAllocs, baseBytes := measure(false)
	allocs, bytes := measure(true)
	allocs, bytes = allocs-baseAllocs, bytes-baseBytes
	t.Logf("blocked Do: %.3f allocs, %.1f B (harness baseline %.3f allocs, %.1f B)", allocs, bytes, baseAllocs, baseBytes)
	const slack = 0.05
	if allocs > 2+slack || bytes > 256 {
		t.Fatalf("blocked Do: %.3f allocs / %.1f B, budget 2 / 256 B", allocs, bytes)
	}
}

func TestBudgetIdleKeysCostNothing(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	op := mustOperation(t, rt, "k",
		func(_ context.Context, s string) (int, error) { return 0, nil },
		Policy[string]{Effect: Pure, SerializeKey: func(s string) string { return s }})
	const n = 100_000
	keys := make([]string, n)
	for i := range keys {
		keys[i] = "key-" + strconv.Itoa(i)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for _, k := range keys {
		if _, err := op.Do(context.Background(), k); err != nil {
			t.Fatal(err)
		}
	}
	runtime.GC()
	runtime.GC() // a second cycle empties the entry pool's victim cache
	runtime.ReadMemStats(&after)
	if live := len(rt.keys.m); live != 0 {
		t.Fatalf("%d key entries survive release, want 0", live)
	}
	grown := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	if perKey := float64(grown) / n; perKey >= 1 {
		t.Fatalf("idle keys retain %.2f B each (%d B total), budget 0", perKey, grown)
	}
	runtime.KeepAlive(keys)
}

// TestBudgetNoGoroutinesPerDo blocks 1000 calls on keys and resources and
// counts goroutines: exactly one per caller, none added by Herta (Δ10).
func TestBudgetNoGoroutinesPerDo(t *testing.T) {
	rt := mustRuntime(t, ResourceSpec{Name: "r", Capacity: 1})
	release := newGate()
	op := mustOperation(t, rt, "g",
		func(context.Context, int) (int, error) { release.wait(); return 0, nil },
		Policy[int]{Effect: Pure, Resources: []Requirement{{Name: "r", Units: 1}},
			SerializeKey: func(i int) string { return strconv.Itoa(i % 2) }})
	const callers = 1000
	base := runtime.NumGoroutine()
	var wg sync.WaitGroup
	for i := range callers {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = op.Do(context.Background(), i) }()
	}
	waitFor(t, func() bool { return rt.Stats().Admitted == callers && blockedCalls(rt) == callers-1 })
	if got := runtime.NumGoroutine() - base; got != callers {
		t.Fatalf("%d goroutines for %d blocked calls, want exactly %d", got, callers, callers)
	}
	release.open()
	wg.Wait()
}

func TestBudgetUncontendedLatency(t *testing.T) {
	if raceEnabled || testing.Short() {
		t.Skip("latency is meaningful only without -race and -short")
	}
	const budget, calls = 150 * time.Nanosecond, 100_000
	op := uncontendedOp(t, 0, nil)
	ctx := context.Background()
	best := time.Duration(1 << 62)
	for range 5 { // best of 5 rejects scheduler noise from concurrent load
		start := time.Now()
		for i := range calls {
			_, _ = op.Do(ctx, i)
		}
		best = min(best, time.Since(start)/calls)
	}
	t.Logf("uncontended Do: %v/op (best of 5)", best)
	if best > budget {
		t.Fatalf("uncontended Do: %v/op, budget %v", best, budget)
	}
}

// queued reads a budget's exact waiter count.
func queued(r *Resource) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.q.len
}

// blockedCalls counts every call parked on a key or a budget of rt.
func blockedCalls(rt *Runtime) int {
	rt.keys.mu.Lock()
	n := 0
	for _, e := range rt.keys.m {
		n += e.q.len
	}
	rt.keys.mu.Unlock()
	for _, r := range rt.budgets {
		n += queued(r)
	}
	return n
}

// waitFor polls cond until it holds, failing the test after 5 s.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); !cond(); runtime.Gosched() {
		if time.Now().After(deadline) {
			t.Fatal("condition not reached within 5s")
		}
	}
}
