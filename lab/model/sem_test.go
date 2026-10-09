package model_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestBudgetSemaphore isolates the budget: one resource, no keys, no
// retries. Random acquire/try/cancel/release sequences with random weights
// are checked step by step against model.Sem, so grants, FIFO order,
// head-of-line blocking and unit accounting all have to match exactly.
func TestBudgetSemaphore(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rnd := workload.Rand(workload.Seed(t))
		for range workload.Scale(200, 3000) {
			capacity := 1 + rnd.Int64N(6)
			sc := budgetScenario(rnd, capacity, 60)
			workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
				driveBudget(t, rnd, r, &sc, capacity)
			})
			if t.Failed() {
				return
			}
		}
	})
}

// budgetScenario declares one op per (admission, weight) and a pool of
// calls that hold until released.
func budgetScenario(rnd *rand.Rand, capacity int64, calls int) model.Scenario {
	sc := model.Scenario{Specs: []impl.Spec{{Name: "r", Capacity: capacity}}}
	for _, adm := range []impl.Admission{impl.Wait, impl.Reject} {
		for w := int64(1); w <= capacity; w++ {
			sc.Ops = append(sc.Ops, model.Op{Name: fmt.Sprintf("%d/%d", adm, w), Effect: impl.Pure,
				Admission: adm, Resources: []impl.Req{{Name: "r", Units: w}}})
		}
	}
	for range calls {
		sc.Calls = append(sc.Calls, model.Call{Op: rnd.IntN(len(sc.Ops)), Script: model.Script{{Block: model.BlockHold}}})
	}
	return sc
}

func driveBudget(t *testing.T, rnd *rand.Rand, r *workload.Rig, sc *model.Scenario, capacity int64) {
	sem := &model.Sem{Cap: capacity}
	want := make([]model.Phase, len(sc.Calls))
	class := make([]impl.Class, len(sc.Calls))
	weight := func(i int) int64 { return sc.Ops[sc.Calls[i].Op].Resources[0].Units }
	grant := func(ids []int) {
		for _, id := range ids {
			want[id] = model.Holding
		}
	}
	next := 0
	for step := 0; step < 2*len(sc.Calls); step++ {
		holding, parked := ids(want, model.Holding), ids(want, model.Parked)
		var did string
		switch n := rnd.IntN(10); {
		case n < 3 && len(holding) > 0:
			id := holding[rnd.IntN(len(holding))]
			r.Release(id)
			want[id] = model.Done
			grant(sem.Release(weight(id)))
			did = fmt.Sprintf("release c%d", id)
		case n < 4 && len(parked) > 0:
			id := parked[rnd.IntN(len(parked))]
			r.Cancel(id)
			want[id], class[id] = model.Done, impl.ClassCanceled
			grant(sem.Cancel(id))
			did = fmt.Sprintf("cancel c%d", id)
		case next < len(sc.Calls):
			id := next
			next++
			r.Start(id)
			switch {
			case sc.Ops[sc.Calls[id].Op].Admission == impl.Reject:
				want[id] = model.Holding
				if !sem.TryAcquire(weight(id)) {
					want[id], class[id] = model.Done, impl.ClassOverloaded
				}
			case sem.Acquire(id, weight(id)):
				want[id] = model.Holding
			default:
				want[id] = model.Parked
			}
			did = fmt.Sprintf("start c%d (op %s)", id, sc.Ops[sc.Calls[id].Op].Name)
		default:
			return
		}
		v := r.View()
		for i := range want {
			if v.Phases[i] != want[i] || (want[i] == model.Done && v.Results[i].Class != class[i]) {
				t.Fatalf("after %s: c%d is %v/%v, model says %v/%v (queue %v, used %d/%d)\n%s",
					did, i, v.Phases[i], v.Results[i].Class, want[i], class[i], sem.Queued(), sem.Used, capacity, model.Format(*sc))
			}
		}
	}
}

func ids(phases []model.Phase, p model.Phase) []int {
	var out []int
	for i, x := range phases {
		if x == p {
			out = append(out, i)
		}
	}
	return out
}

// TestBudgetUnderContention checks the budget's safety on a real
// scheduler, where no model can predict the interleaving: the trace of
// units in flight never exceeds capacity, every unit comes back, and
// nothing hangs, under weighted, cancelling, oversubscribed load.
func TestBudgetUnderContention(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rnd := workload.Rand(workload.Seed(t))
		const capacity = 5
		spec := impl.Spec{Name: "r", Capacity: capacity}
		rt, err := im.NewRuntime(spec)
		if err != nil {
			t.Fatal(err)
		}
		mon := workload.NewMonitor(spec)
		var ops []impl.Op
		for w := int64(1); w <= capacity; w++ {
			reqs := []impl.Req{{Name: "r", Units: w}}
			op, err := rt.NewOp("w", impl.Policy{Effect: impl.Pure, Resources: reqs}, func(ctx context.Context, in int) (int, error) {
				defer mon.Enter(reqs, "")()
				time.Sleep(time.Duration(in) * time.Microsecond)
				return in, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			ops = append(ops, op)
		}
		workers, perWorker := 64, workload.Scale(200, 5000)
		seeds := make([]uint64, workers)
		for i := range seeds {
			seeds[i] = rnd.Uint64()
		}
		workload.Within(t, 60*time.Second, "contended budget", func() {
			var wg sync.WaitGroup
			for w := range workers {
				wg.Go(func() {
					wr := workload.Rand(seeds[w])
					for range perWorker {
						ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wr.IntN(300))*time.Microsecond)
						ops[wr.IntN(len(ops))].Do(ctx, wr.IntN(50))
						cancel()
					}
				})
			}
			wg.Wait()
		})
		mon.Check(t)
		if p := mon.Peak("r"); p > capacity {
			t.Errorf("peak %d units in flight, capacity %d", p, capacity)
		}
		// Conservation: with nothing in flight the whole budget is free.
		full, _ := rt.NewOp("full", impl.Policy{Effect: impl.Pure, Admission: impl.Reject,
			Resources: []impl.Req{{Name: "r", Units: capacity}}}, func(context.Context, int) (int, error) { return 0, nil })
		if _, err := full.Do(context.Background(), 0); err != nil {
			t.Errorf("full-capacity Reject after the storm: %v (units leaked)", err)
		}
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Fatal(err)
		}
		t.Logf("%d calls; peak %d/%d units", workers*perWorker, mon.Peak("r"), capacity)
	})
}
