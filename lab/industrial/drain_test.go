package industrial_test

import (
	"context"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestDrainUnderLoad saturates a budget of 8 units (or 16 keys) with
// closed-loop callers whose handler takes 2ms, so nearly all of them are
// parked, then calls Shutdown. §6: Shutdown withdraws every waiter at once
// and returns when the handlers already running have finished, so a drain
// costs about one service time however long the queue is; when it returns
// nil, Finished == Admitted already (Δ2). Every caller must then return,
// and no goroutine may outlive the runtime.
func TestDrainUnderLoad(t *testing.T) {
	callers, reps := workload.Scale(256, 1024), workload.Scale(3, 20)
	const service = 2 * time.Millisecond
	for _, on := range []string{"budget", "key"} {
		t.Run(on, func(t *testing.T) {
			workload.Each(t, func(t *testing.T, im impl.Impl) {
				var drains, exits []float64
				lagged := 0
				for range reps {
					drain, exit, lag := drainOnce(t, im, on == "key", callers, service)
					drains, exits = append(drains, ms(drain)), append(exits, ms(exit))
					if lag {
						lagged++
					}
				}
				slices.Sort(drains)
				slices.Sort(exits)
				m := metrics{
					"callers": float64(callers), "reps": float64(reps), "service_ms": ms(service),
					"drain_p50_ms": drains[len(drains)/2], "drain_max_ms": drains[len(drains)-1],
					"exit_p50_ms": exits[len(exits)/2], "exit_max_ms": exits[len(exits)-1],
					"stats_lag_reps": float64(lagged),
				}
				record(t, im, m)
				budget(t, im, workload.D2, "drains with Finished < Admitted", m["stats_lag_reps"], 0)
				if m["drain_max_ms"] > ms(service)+25 {
					t.Errorf("drain took %.1fms, more than one %v service time (+25ms)", m["drain_max_ms"], service)
				}
			})
		})
	}
}

// drainOnce runs one saturated runtime and shuts it down under load. It
// returns how long Shutdown took, how long until every caller had
// returned, and whether Stats lagged the drain.
func drainOnce(t *testing.T, im impl.Impl, keyed bool, callers int, service time.Duration) (drain, exit time.Duration, lag bool) {
	t.Helper()
	baseG := workload.SettledGoroutines()
	capacity := int64(8)
	p := impl.Policy{Effect: impl.Idempotent, Resources: []impl.Req{{Name: "r", Units: 1}}}
	if keyed {
		capacity = 1 << 30
		keys := make([]string, 16)
		for i := range keys {
			keys[i] = "k" + strconv.Itoa(i)
		}
		p.Key = func(in int) string { return keys[in%len(keys)] }
	}
	rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: capacity})
	if err != nil {
		t.Fatal(err)
	}
	op, err := rt.NewOp("work", p, func(ctx context.Context, in int) (int, error) {
		time.Sleep(service)
		return in, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for id := range callers {
		wg.Go(func() {
			for {
				if _, err := op.Do(context.Background(), id); err != nil {
					if c := im.Classify(err); c != impl.ClassStopping {
						t.Errorf("caller %d: %v (%v)", id, err, c)
					}
					return
				}
			}
		})
	}
	waitFor(t, 10*time.Second, "every caller inside Do", func() bool {
		st := rt.Stats()
		return st.Admitted-st.Finished == uint64(callers)
	})
	time.Sleep(10 * time.Millisecond) // the queue reaches its steady shape

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := workload.Now()
	if err := rt.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	drain = workload.Since(start)
	st := rt.Stats()
	workload.Within(t, 5*time.Second, "callers after Shutdown", wg.Wait)
	exit = workload.Since(start)
	workload.NoLeak(t, baseG, 0, 2*time.Second)
	return drain, exit, st.Finished != st.Admitted
}
