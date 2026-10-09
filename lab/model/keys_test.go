package model_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestKeyLocker isolates the key locker: keyed operations that claim no
// resources. Random start/release/cancel sequences over a few keys are
// checked step by step against model.Keys: exclusion, FIFO hand-off, and
// cancellation of queued waiters. A keyed Reject call parks in v0.2 and
// fails fast with Rejected counted under the contract (D1).
func TestKeyLocker(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rnd := workload.Rand(workload.Seed(t))
		fixed := workload.Fixed(im, workload.D1)
		for range workload.Scale(200, 3000) {
			sc := keyScenario(rnd, 60)
			workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
				driveKeys(t, rnd, r, &sc, fixed)
			})
			if t.Failed() {
				return
			}
		}
	})
}

func keyScenario(rnd *rand.Rand, calls int) model.Scenario {
	sc := model.Scenario{
		Specs: []impl.Spec{{Name: "unused", Capacity: 1}},
		Ops: []model.Op{
			{Name: "wait", Effect: impl.Idempotent, Keyed: true},
			{Name: "reject", Effect: impl.Idempotent, Keyed: true, Admission: impl.Reject},
		},
	}
	for range calls {
		sc.Calls = append(sc.Calls, model.Call{Op: rnd.IntN(4) / 3, Key: "k" + strconv.Itoa(rnd.IntN(3)),
			Script: model.Script{{Block: model.BlockHold}}})
	}
	return sc
}

func driveKeys(t *testing.T, rnd *rand.Rand, r *workload.Rig, sc *model.Scenario, fixed bool) {
	var keys model.Keys
	want := make([]model.Phase, len(sc.Calls))
	class := make([]impl.Class, len(sc.Calls))
	var rejected uint64
	handoff := func(id int) {
		if id >= 0 {
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
			handoff(keys.Release(sc.Calls[id].Key))
			did = fmt.Sprintf("release c%d", id)
		case n < 4 && len(parked) > 0:
			id := parked[rnd.IntN(len(parked))]
			r.Cancel(id)
			want[id], class[id] = model.Done, impl.ClassCanceled
			keys.Cancel(id, sc.Calls[id].Key)
			did = fmt.Sprintf("cancel c%d", id)
		case next < len(sc.Calls):
			id := next
			next++
			r.Start(id)
			c := sc.Calls[id]
			switch {
			case fixed && c.Op == 1 && keys.Held(c.Key):
				want[id], class[id] = model.Done, impl.ClassOverloaded
				rejected++
			case keys.Acquire(id, c.Key):
				want[id] = model.Holding
			default:
				want[id] = model.Parked
			}
			did = fmt.Sprintf("start c%d (%s %s)", id, sc.Ops[c.Op].Name, c.Key)
		default:
			return
		}
		v := r.View()
		for i := range want {
			if v.Phases[i] != want[i] || (want[i] == model.Done && v.Results[i].Class != class[i]) {
				t.Fatalf("after %s: c%d is %v/%v, model says %v/%v\n%s",
					did, i, v.Phases[i], v.Results[i].Class, want[i], class[i], model.Format(*sc))
			}
		}
		if v.Stats.Rejected != rejected {
			t.Fatalf("after %s: Rejected=%d, model says %d (D1, §7)", did, v.Stats.Rejected, rejected)
		}
	}
}

// TestKeyExclusionUnderContention: on a real scheduler, a key never has
// two holders, however many callers fight over it, and nobody hangs.
func TestKeyExclusionUnderContention(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rnd := workload.Rand(workload.Seed(t))
		rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 1})
		if err != nil {
			t.Fatal(err)
		}
		mon := workload.NewMonitor()
		key := func(in int) string { return "k" + strconv.Itoa(in%4) }
		op, err := rt.NewOp("keyed", impl.Policy{Effect: impl.Idempotent, Key: key}, func(ctx context.Context, in int) (int, error) {
			defer mon.Enter(nil, key(in))()
			time.Sleep(time.Duration(in%7) * time.Microsecond)
			return in, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		workers, perWorker := 64, workload.Scale(200, 5000)
		base := rnd.Uint64()
		workload.Within(t, 60*time.Second, "contended keys", func() {
			var wg sync.WaitGroup
			for w := range workers {
				wg.Go(func() {
					wr := workload.Rand(base + uint64(w))
					for range perWorker {
						ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wr.IntN(200))*time.Microsecond)
						op.Do(ctx, wr.IntN(1000))
						cancel()
					}
				})
			}
			wg.Wait()
		})
		mon.Check(t)
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Fatal(err)
		}
	})
}

// TestKeyReclamation: idle keys cost nothing (§4.2, §10). Retained heap
// must not grow with the number of distinct keys ever used. A warm-up burst
// first pays the costs that scale with peak concurrency instead (dead
// goroutine descriptors the scheduler keeps for reuse, map capacity); that
// one-off cost is reported, and a second identical burst must add nothing.
func TestKeyReclamation(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 1})
		if err != nil {
			t.Fatal(err)
		}
		hold := make(chan struct{})
		var entered sync.WaitGroup
		op, err := rt.NewOp("keyed", impl.Policy{Effect: impl.Idempotent, Key: strconv.Itoa},
			func(ctx context.Context, in int) (int, error) {
				if in < 0 {
					entered.Done()
					<-hold
				}
				return in, nil
			})
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		burstN, serialN := workload.Scale(5_000, 50_000), workload.Scale(100_000, 2_000_000)
		burst := func(offset int) {
			hold = make(chan struct{})
			entered.Add(burstN)
			var done sync.WaitGroup
			for i := range burstN {
				done.Go(func() { op.Do(ctx, -1-offset-i) })
			}
			entered.Wait()
			close(hold)
			done.Wait()
		}
		const slack = 256 << 10

		cold := workload.LiveHeap()
		burst(0)
		base := workload.LiveHeap()
		for i := range serialN {
			op.Do(ctx, i)
		}
		afterSerial := workload.LiveHeap()
		burst(burstN)
		afterBurst := workload.LiveHeap()

		t.Logf("one-off cost of %d concurrent keys: %d B (%.0f B/key); %d distinct serial keys: %+d B; second burst: %+d B",
			burstN, base-cold, float64(base-cold)/float64(burstN), serialN, int64(afterSerial)-int64(base), int64(afterBurst)-int64(base))
		if afterSerial > base+slack {
			t.Errorf("heap grew %d B over %d distinct keys: idle keys are retained", afterSerial-base, serialN)
		}
		if afterBurst > base+max(slack, (base-cold)/10) {
			t.Errorf("a second burst of %d keys retained %d B more: retention is cumulative", burstN, afterBurst-base)
		}
		if err := rt.Shutdown(ctx); err != nil {
			t.Fatal(err)
		}
	})
}
