package industrial_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestMemoryPerWaiter parks n callers behind one held budget unit, or one
// held key, and charges the runtime for what they cost beyond n parked
// goroutines. §10: a blocked acquire costs ≤ 2 allocations and ≤ 256 B of
// heap, excluding its goroutine's stack, and starts no goroutine (Δ10).
// Then every waiter's context is cancelled at once, and the withdrawal
// must return all n callers with context.Canceled.
//
// Two baselines split the bill. Against goroutines parked in a two-way
// select (what any waiter that honours cancellation must do) the remainder
// is the library's own cost, which §10 budgets. Against goroutines parked
// on their context alone it is the process cost: the library's plus the
// Go scheduler's second select record (a sudog, ~110 B, 1 allocation once
// the scheduler's cache runs dry), reported but not budgeted.
func TestMemoryPerWaiter(t *testing.T) {
	n := workload.Scale(1_000, 10_000)
	never := make(chan struct{})
	oneWay := func(ctx context.Context, _ int) { <-ctx.Done() }
	twoWay := func(ctx context.Context, _ int) {
		select {
		case <-ctx.Done():
		case <-never:
		}
	}
	all := func() bool { return true }
	for _, on := range []string{"budget", "key"} {
		t.Run(on, func(t *testing.T) {
			workload.Each(t, func(t *testing.T, im impl.Impl) {
				parkAll(t, n, oneWay, all).release() // warm-up: grows the scheduler's tables once
				one := parkAll(t, n, oneWay, all)
				one.release()
				two := parkAll(t, n, twoWay, all)
				two.release()

				op, held, st := heldOp(t, im, on == "key")
				var canceled atomic.Int64
				p := parkAll(t, n, func(ctx context.Context, i int) {
					if _, err := op.Do(ctx, i); im.Classify(err) == impl.ClassCanceled {
						canceled.Add(1)
					}
				}, func() bool { return st().Admitted == uint64(n)+1 })
				withdraw := p.release()
				held()

				heap, stack, allocs, gs := p.parked.per(p.base, n)
				h1, s1, a1, g1 := one.parked.per(one.base, n)
				h2, _, a2, _ := two.parked.per(two.base, n)
				m := metrics{
					"waiters": float64(n), "heap_B_per_waiter": heap - h2, "allocs_per_waiter": allocs - a2,
					"process_heap_B_per_waiter": heap - h1, "process_allocs_per_waiter": allocs - a1,
					"stack_B_per_waiter": stack - s1, "goroutines_per_waiter": gs - g1,
					"withdraw_ms": ms(withdraw), "canceled_frac": float64(canceled.Load()) / float64(n),
				}
				record(t, im, m)
				budget(t, im, workload.D10, "heap B per waiter", m["heap_B_per_waiter"], 256)
				// 2.5, not 2: the baselines' sudog-cache refills vary between phases.
				// Over 80 runs recon measured 1.16-2.17 (median 1.61) for its exact 2
				// (recon's TestBudgetBlockedWaiter pins the 2); v0.2 measures ~10-15.
				budget(t, im, workload.D10, "allocs per waiter", m["allocs_per_waiter"], 2.5)
				budget(t, im, workload.D10, "goroutines per waiter", m["goroutines_per_waiter"], 0.01)
				if canceled.Load() != int64(n) {
					t.Errorf("%d of %d cancelled waiters returned context.Canceled", canceled.Load(), n)
				}
			})
		})
	}
}

// heldOp builds a runtime whose Wait operation is blocked: one call holds
// the budget's only unit (or, keyed, the one key every call uses) inside
// its handler until release is called, which also drains the runtime.
func heldOp(t *testing.T, im impl.Impl, keyed bool) (op impl.Op, release func(), stats func() impl.Stats) {
	t.Helper()
	capacity := int64(1)
	p := impl.Policy{Effect: impl.Idempotent}
	if keyed {
		capacity = 1 << 30
		p.Key = func(int) string { return "hot" }
	}
	p.Resources = []impl.Req{{Name: "r", Units: 1}}
	rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: capacity})
	if err != nil {
		t.Fatal(err)
	}
	gate, entered := make(chan struct{}), make(chan struct{})
	op, err = rt.NewOp("held", p, func(ctx context.Context, in int) (int, error) {
		if in < 0 {
			close(entered)
			<-gate
		}
		return in, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	holder := make(chan struct{})
	go func() { defer close(holder); op.Do(context.Background(), -1) }()
	<-entered
	return op, func() {
		close(gate)
		<-holder
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := rt.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
	}, rt.Stats
}

// parked is n goroutines blocked in do, measured once they settled.
type parked struct {
	base, parked memSnap
	cancel       context.CancelFunc
	wg           *sync.WaitGroup
}

// parkAll starts n goroutines running do with one shared cancellable
// context, waits for admitted() and for the goroutine count to settle,
// and snapshots memory before and after. The harness and the runtime run
// through this one function, so their own costs cancel out.
func parkAll(t *testing.T, n int, do func(ctx context.Context, i int), admitted func() bool) parked {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	p := parked{cancel: cancel, wg: &sync.WaitGroup{}}
	var started atomic.Int64
	p.base = snapshot()
	for i := range n {
		p.wg.Go(func() {
			started.Add(1)
			do(ctx, i)
		})
	}
	waitFor(t, 30*time.Second, "waiters admitted", func() bool { return started.Load() == int64(n) && admitted() })
	workload.SettledGoroutines()
	time.Sleep(20 * time.Millisecond) // the last admitted callers reach their park
	p.parked = snapshot()
	return p
}

// release cancels every waiter and returns how long they took to return.
func (p parked) release() time.Duration {
	start := workload.Now()
	p.cancel()
	p.wg.Wait()
	return workload.Since(start)
}
