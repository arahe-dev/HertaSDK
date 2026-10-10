package assume

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestKeyWaitersHoldNoUnits
//
// Assumption: the key is acquired before resources, so a call queued on a
// busy key holds zero resource units (§4 step 2).
//
// Why it matters: the catalogue serializes writes per brand on the shared
// db-write budget (4). If key waiters held units, one hot brand would
// starve events and handoff of the whole budget.
//
// Verdict: with one brand-"k" call holding 1 of 2 units and five more
// queued on "k", a call for brand "k2" still gets the second unit at once,
// and a third brand then parks: exactly one unit was in use for "k".
func TestKeyWaitersHoldNoUnits(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		op := op1("keyed", impl.Wait, 1)
		op.Keyed = true
		sc := model.Scenario{
			Specs: []impl.Spec{{Name: "r", Capacity: 2}},
			Ops:   []model.Op{op},
			Calls: append(calls(6, 0, "k", held), model.Call{Key: "k2", Script: held}, model.Call{Key: "k3", Script: held}),
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			start(r, 0, 1, 2, 3, 4, 5)
			expect(t, r, model.Holding, 0)
			expect(t, r, model.Parked, 1, 2, 3, 4, 5)
			start(r, 6, 7)
			expect(t, r, model.Holding, 6)
			expect(t, r, model.Parked, 7)
		})
	})
}

// TestMultiResourceHoldAndWait
//
// Assumption: a call waiting for a later resource (in name order) keeps the
// earlier ones it already acquired (§4.3, the one stated exception to A3).
//
// Why it matters: idle-but-held capacity is invisible in Stats. It is
// deadlock-free only because every call acquires in the same global order,
// and v1.0 may replace it with all-or-nothing acquisition.
//
// Verdict: with b held, a call needing {a, b} takes a and parks on b; a
// call needing only a then parks although nothing is running on a.
func TestMultiResourceHoldAndWait(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		two := func(name string, res ...string) model.Op {
			op := model.Op{Name: name, Effect: impl.Pure}
			for _, r := range res {
				op.Resources = append(op.Resources, impl.Req{Name: r, Units: 1})
			}
			return op
		}
		sc := model.Scenario{
			Specs: []impl.Spec{{Name: "a", Capacity: 1}, {Name: "b", Capacity: 1}},
			// Declared b-first: acquisition order must still be a, then b.
			Ops:   []model.Op{two("b", "b"), two("ba", "b", "a"), two("a", "a")},
			Calls: []model.Call{{Op: 0, Script: held}, {Op: 1, Script: held}, {Op: 2, Script: held}},
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			start(r, 0, 1, 2)
			expect(t, r, model.Holding, 0)
			expect(t, r, model.Parked, 1, 2)
			r.Release(0)
			expect(t, r, model.Holding, 1)
			expect(t, r, model.Parked, 2)
		})
	})
}

// TestDoneCallerTakesNothing
//
// Assumption: a caller whose context is already done takes no key, in
// either mode, and queues for no units (CONTRACT Amendment 1). v0.2 gets
// this from x/sync's Acquire, which fails first on a done context; only a
// Reject claim (TryAcquire) ignores the context.
//
// Why it matters: a request that has already timed out upstream must not
// hold a brand's key or a db-write unit, and must not be counted as
// overload: it would inflate shed metrics and could trip load shedding.
//
// Verdict: with key "k" and the only unit held, done callers get
// context.Canceled for a held key (Reject), a free key (Reject), and a
// Wait claim; only an unkeyed Reject call reports overload, and Rejected
// counts exactly that one.
func TestDoneCallerTakesNothing(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		keyedWait, keyedReject, reject := op1("kw", impl.Wait, 1), op1("kr", impl.Reject, 1), op1("r", impl.Reject, 1)
		keyedWait.Keyed, keyedReject.Keyed = true, true
		done := func(op int, key string) model.Call {
			return model.Call{Op: op, Key: key, Precancel: true, Script: held}
		}
		sc := model.Scenario{
			Specs: []impl.Spec{{Name: "r", Capacity: 1}},
			Ops:   []model.Op{keyedWait, keyedReject, reject},
			Calls: []model.Call{{Op: 0, Key: "k", Script: held}, done(1, "k"), done(1, "k2"), done(2, ""), done(0, "k3")},
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			start(r, 0, 1, 2, 3, 4)
			expect(t, r, model.Holding, 0)
			expectResult(t, r, 1, impl.ClassCanceled)
			expectResult(t, r, 2, impl.ClassCanceled)
			expectResult(t, r, 3, impl.ClassOverloaded)
			expectResult(t, r, 4, impl.ClassCanceled)
			if s := r.Runtime().Stats(); s.Rejected != 1 {
				t.Fatalf("Rejected %d, want 1 (only the unkeyed Reject overload)", s.Rejected)
			}
			r.Release(0)
		})
	})
}

// TestNoDeadlockUnderRandomContention
//
// Assumption: acquiring keys, then resources in one global name order,
// makes every workload deadlock-free, whatever order operations declare
// their resources in.
//
// Why it matters: §4.3 hold-and-wait is safe only under this assumption;
// a deadlock here is a production hang with no error and no timeout.
//
// Verdict: random operations over 4 resources (claims declared in random
// order, random units, some keyed) driven by 64 goroutines with no caller
// deadlines all finish within the watchdog, with capacity and key
// exclusion intact throughout.
func TestNoDeadlockUnderRandomContention(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rnd := workload.Rand(workload.Seed(t))
		specs := []impl.Spec{{Name: "a", Capacity: 1}, {Name: "b", Capacity: 2}, {Name: "c", Capacity: 3}, {Name: "d", Capacity: 1}}
		rt, err := im.NewRuntime(specs...)
		if err != nil {
			t.Fatal(err)
		}
		mon := workload.NewMonitor(specs...)
		var ops []impl.Op
		for i := range 12 {
			var reqs []impl.Req
			for _, j := range rnd.Perm(len(specs)) {
				if rnd.IntN(2) == 0 {
					reqs = append(reqs, impl.Req{Name: specs[j].Name, Units: 1 + rnd.Int64N(specs[j].Capacity)})
				}
			}
			p := impl.Policy{Effect: impl.Pure, Resources: reqs}
			keyed := rnd.IntN(3) == 0
			key := func(in int) string { return "k" + strconv.Itoa(in%3) }
			if keyed {
				p.Key = key
			}
			op, err := rt.NewOp(fmt.Sprint("op", i), p, func(ctx context.Context, in int) (int, error) {
				k := ""
				if keyed {
					k = key(in)
				}
				defer mon.Enter(reqs, k)()
				time.Sleep(time.Duration(in%5) * 10 * time.Microsecond)
				return in, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			ops = append(ops, op)
		}
		base := rnd.Uint64()
		workload.Within(t, 60*time.Second, "random multi-resource workload", func() {
			var wg sync.WaitGroup
			for w := range 64 {
				wg.Go(func() {
					wr := workload.Rand(base + uint64(w))
					for range workload.Scale(150, 3000) {
						ops[wr.IntN(len(ops))].Do(context.Background(), wr.IntN(100))
					}
				})
			}
			wg.Wait()
		})
		mon.Check(t)
		rt.Shutdown(context.Background())
	})
}

// TestReleaseOrderUnobservable
//
// Assumption: resources are released in reverse acquisition order (§4
// step 3), and that order matters to callers.
//
// Why it matters: it is written into the contract, so it needs either a
// test or an argument.
//
// Verdict: the order is not observable at quiescence. Releasing a holder
// of {a, b} admits a waiter for {a, b} completely in the same step either
// way (the mutation "release in acquisition order" is equivalent, as the
// v0.2 review's M03 found). Reverse order is an internal property: it is
// what lets lab/model treat a release cascade as deterministic, and it is
// checked there indirectly. This test pins the observable half.
func TestReleaseOrderUnobservable(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		ab := model.Op{Name: "ab", Effect: impl.Pure, Resources: []impl.Req{{Name: "a", Units: 1}, {Name: "b", Units: 1}}}
		sc := model.Scenario{
			Specs: []impl.Spec{{Name: "a", Capacity: 1}, {Name: "b", Capacity: 1}},
			Ops:   []model.Op{ab},
			Calls: calls(2, 0, "", held),
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			start(r, 0, 1)
			expect(t, r, model.Parked, 1)
			r.Release(0)
			expect(t, r, model.Holding, 1)
		})
	})
}
