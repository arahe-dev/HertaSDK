package industrial_test

import (
	"context"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestManyUniqueKeys sends n calls through 64 workers, each on a key never
// used before, and measures what the keys leave behind. §4.2 and §10: an
// idle key costs 0 B, so with the runtime still alive the live heap after
// the run must be back where it started, and a keyed call allocates at
// most 1 more time than an unkeyed one. Also reported: keyed and unkeyed
// throughput, the peak heap in use above the starting live heap, and the
// heap per key held concurrently.
func TestManyUniqueKeys(t *testing.T) {
	n, held := workload.Scale(100_000, 1_000_000), workload.Scale(1_000, 10_000)
	keys := make([]string, n) // built once, so key formatting is no one's cost
	for i := range keys {
		keys[i] = "brand-" + strconv.Itoa(i)
	}
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		plain := pump(t, im, keys, nil)
		keyed := pump(t, im, keys, func(in int) string { return keys[in] })
		perHeld := heldKeyCost(t, im, keys[:held])
		m := metrics{
			"keys": float64(n), "keyed_calls_per_s": keyed.rate, "unkeyed_calls_per_s": plain.rate,
			"extra_allocs_per_keyed_call": keyed.allocs - plain.allocs,
			"retained_KB":                 keyed.retained / 1024, "idle_B_per_key": keyed.retained / float64(n),
			"peak_extra_heap_MB": keyed.peakHeap, "held_keys": float64(held), "heap_B_per_held_key": perHeld,
		}
		record(t, im, m)
		budget(t, im, workload.D10, "extra allocations per keyed call", m["extra_allocs_per_keyed_call"], 1)
		// §10's idle-key budget is recon's. v0.2 is measured: in 5 of ~90 quick
		// runs it retained 160-284 KB after the churn (recon: always |x| < 25 KB).
		budget(t, im, workload.D10, "KB retained by idle keys (§10: 0 B)", m["retained_KB"], 64)
	})
}

type pumped struct {
	rate, allocs, retained, peakHeap float64
}

// pump drives one call per key through 64 workers, against a fresh
// runtime that stays alive until the retained heap has been read.
func pump(t *testing.T, im impl.Impl, keys []string, key func(int) string) pumped {
	t.Helper()
	const workers = 64
	rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: workers})
	if err != nil {
		t.Fatal(err)
	}
	p := impl.Policy{Effect: impl.Idempotent, Resources: []impl.Req{{Name: "r", Units: 1}}, Key: key}
	op, err := rt.NewOp("pump", p, func(_ context.Context, in int) (int, error) { return in, nil })
	if err != nil {
		t.Fatal(err)
	}
	before := snapshot()
	pk := watch(func() int64 { return 0 })
	start := workload.Now()
	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			for i := w; i < len(keys); i += workers {
				if _, err := op.Do(context.Background(), i); err != nil {
					t.Errorf("call %d: %v", i, err)
					return
				}
			}
		})
	}
	wg.Wait()
	took := workload.Since(start)
	_, peak := pk.stop()
	after := snapshot()
	runtime.KeepAlive(op)
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
	n := float64(len(keys))
	return pumped{
		rate: n / took.Seconds(), allocs: float64(after.mallocs-before.mallocs) / n,
		retained: float64(after.heap) - float64(before.heap), peakHeap: peak - float64(before.heap)/(1<<20),
	}
}

// heldKeyCost is the live heap per key while len(keys) calls each hold a
// distinct key inside their handler, beyond the same calls unkeyed.
func heldKeyCost(t *testing.T, im impl.Impl, keys []string) float64 {
	t.Helper()
	cost := func(key func(int) string) float64 {
		rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: int64(len(keys))})
		if err != nil {
			t.Fatal(err)
		}
		var entered atomic.Int64
		p := impl.Policy{Effect: impl.Idempotent, Resources: []impl.Req{{Name: "r", Units: 1}}, Key: key}
		op, err := rt.NewOp("hold", p, func(ctx context.Context, in int) (int, error) {
			entered.Add(1)
			<-ctx.Done()
			return in, ctx.Err()
		})
		if err != nil {
			t.Fatal(err)
		}
		h := parkAll(t, len(keys), func(ctx context.Context, i int) { op.Do(ctx, i) },
			func() bool { return entered.Load() == int64(len(keys)) })
		h.release()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := rt.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
		heap, _, _, _ := h.parked.per(h.base, len(keys))
		return heap
	}
	cost(nil) // warm-up: grows the scheduler's tables once
	return cost(func(in int) string { return keys[in] }) - cost(nil)
}
