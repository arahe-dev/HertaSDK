package assume

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestCapacityExact
//
// Assumption: a budget of capacity C runs exactly C units of work at once,
// however oversubscribed: never more, and — while work is queued — never
// fewer.
//
// Why it matters: capacity is the product. The renderer budget (8) stands
// between Mirai and a billed provider; one extra unit is an outage, one
// missing unit is lost throughput.
//
// Verdict: deterministic half — 1000 Wait calls on capacity 8 leave exactly
// 8 holding and 992 parked, and every release admits exactly one more.
// Real-scheduler half — 256 goroutines hammering capacity 8 reach a peak
// of exactly 8 (monitored inside the handler).
func TestCapacityExact(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		sc := model.Scenario{
			Specs: []impl.Spec{{Name: "r", Capacity: 8}},
			Ops:   []model.Op{op1("w", impl.Wait, 1)},
			Calls: calls(1000, 0, "", held),
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			for i := range sc.Calls {
				r.Start(i)
			}
			for released := 0; released < 50; released++ {
				if h, p := count(r, model.Holding), count(r, model.Parked); h != 8 || p != 992-released {
					t.Fatalf("after %d releases: %d holding, %d parked; want 8 and %d", released, h, p, 992-released)
				}
				r.Release(holders(r)[0])
			}
		})

		spec := impl.Spec{Name: "r", Capacity: 8}
		rt, err := im.NewRuntime(spec)
		if err != nil {
			t.Fatal(err)
		}
		mon := workload.NewMonitor(spec)
		reqs := []impl.Req{{Name: "r", Units: 1}}
		op, _ := rt.NewOp("w", impl.Policy{Effect: impl.Pure, Resources: reqs}, func(ctx context.Context, in int) (int, error) {
			defer mon.Enter(reqs, "")()
			time.Sleep(50 * time.Microsecond)
			return in, nil
		})
		workload.Within(t, 30*time.Second, "oversubscribed budget", func() {
			var wg sync.WaitGroup
			for range 256 {
				wg.Go(func() {
					for range workload.Scale(20, 400) {
						op.Do(context.Background(), 0)
					}
				})
			}
			wg.Wait()
		})
		mon.Check(t)
		if p := mon.Peak("r"); p != 8 {
			t.Errorf("peak %d units in flight under 32× oversubscription, want exactly 8", p)
		}
		rt.Shutdown(context.Background())
	})
}

// TestFIFOAdmission
//
// Assumption: waiters are served strictly first come, first served (§4.1).
//
// Why it matters: FIFO is what bounds a waiter's delay by the queue ahead
// of it; without it a p99 admission-wait SLO cannot be reasoned about.
//
// Verdict: 20 waiters queued in a known order behind one holder of a
// capacity-1 budget are admitted in exactly that order, one per release.
func TestFIFOAdmission(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		sc := model.Scenario{
			Specs: []impl.Spec{{Name: "r", Capacity: 1}},
			Ops:   []model.Op{op1("w", impl.Wait, 1)},
			Calls: calls(21, 0, "", held),
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			for i := range sc.Calls {
				start(r, i) // c(i) is queued before c(i+1) arrives
			}
			for i := range 20 {
				expect(t, r, model.Holding, i)
				expect(t, r, model.Parked, i+1)
				r.Release(i)
			}
		})
	})
}

// TestHeadOfLineBlocking
//
// Assumption: a queued request that does not fit blocks every request
// behind it, even ones that would fit (§4.1, x/sync semantics).
//
// Why it matters: this is what keeps a heavy request from starving, and
// also why a single large waiter can stall a budget with free units. The
// docs once promised "fair wake-up"; this pins what actually happens.
//
// Verdict: with 3 of 4 units held, a 4-unit waiter parks, and a 1-unit
// waiter behind it parks too although 1 unit is free; releasing the holder
// admits the heavy waiter first, and the light one only after it.
func TestHeadOfLineBlocking(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		sc := model.Scenario{
			Specs: []impl.Spec{{Name: "r", Capacity: 4}},
			Ops:   []model.Op{op1("three", impl.Wait, 3), op1("four", impl.Wait, 4), op1("one", impl.Wait, 1)},
			Calls: []model.Call{{Op: 0, Script: held}, {Op: 1, Script: held}, {Op: 2, Script: held}},
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			start(r, 0, 1, 2)
			expect(t, r, model.Holding, 0)
			expect(t, r, model.Parked, 1, 2)
			r.Release(0)
			expect(t, r, model.Holding, 1)
			expect(t, r, model.Parked, 2)
			r.Release(1)
			expect(t, r, model.Holding, 2)
		})
	})
}

// TestRejectFailsWithWaitersQueued
//
// Assumption: Reject fails whenever anyone is queued, even if its units
// are free (§4.1: granted immediately only if it fits and no one waits).
//
// Why it matters: a Reject caller cannot jump a queue, so Reject sheds
// load earlier than "budget full" suggests; overload alarms keyed on
// Throttled fire while capacity is idle.
//
// Verdict: with 1 of 2 units held and a 2-unit waiter queued, a 1-unit
// Reject call fails with ErrOverloaded (Throttled) and counts in Rejected.
func TestRejectFailsWithWaitersQueued(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		sc := model.Scenario{
			Specs: []impl.Spec{{Name: "r", Capacity: 2}},
			Ops:   []model.Op{op1("one", impl.Wait, 1), op1("two", impl.Wait, 2), op1("try", impl.Reject, 1)},
			Calls: []model.Call{{Op: 0, Script: held}, {Op: 1, Script: held}, {Op: 2, Script: held}},
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			start(r, 0, 1, 2)
			expectResult(t, r, 2, impl.ClassOverloaded)
			if o := r.Result(2).Outcome; o != impl.Throttled {
				t.Errorf("overload outcome %v, want Throttled", o)
			}
			if s := r.View().Stats; s.Rejected != 1 || s.Admitted != 3 {
				t.Errorf("stats %+v: want Admitted 3 (overloads pass the gate), Rejected 1", s)
			}
		})
	})
}

// TestHeadCancelReexamines
//
// Assumption: when the head waiter gives up, the waiters behind it are
// re-examined at once (§4.1), not at the next release.
//
// Why it matters: otherwise one cancelled heavy request strands lighter
// waiters behind free capacity until some unrelated call finishes — an
// unbounded stall on a quiet budget.
//
// Verdict: with 3 of 4 units held, a 4-unit head waiter and a 1-unit
// waiter behind it, cancelling the head admits the 1-unit waiter in the
// same step.
func TestHeadCancelReexamines(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		sc := model.Scenario{
			Specs: []impl.Spec{{Name: "r", Capacity: 4}},
			Ops:   []model.Op{op1("three", impl.Wait, 3), op1("four", impl.Wait, 4), op1("one", impl.Wait, 1)},
			Calls: []model.Call{{Op: 0, Script: held}, {Op: 1, Script: held}, {Op: 2, Script: held}},
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			start(r, 0, 1, 2)
			expect(t, r, model.Parked, 1, 2)
			r.Cancel(1)
			expectResult(t, r, 1, impl.ClassCanceled)
			expect(t, r, model.Holding, 0, 2)
		})
	})
}
