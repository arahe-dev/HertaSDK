package industrial_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// metrics is one test's numbers for one implementation, by name.
type metrics map[string]float64

// record logs m and, if LAB_OUT is set, appends it to that file as a JSON
// line once the test has finished, with its verdict.
func record(t *testing.T, im impl.Impl, m metrics) {
	t.Helper()
	var b strings.Builder
	for _, k := range slices.Sorted(maps.Keys(m)) {
		fmt.Fprintf(&b, " %s=%.4g", k, m[k])
	}
	t.Logf("metrics:%s", b.String())
	out := os.Getenv("LAB_OUT")
	if out == "" {
		return
	}
	t.Cleanup(func() {
		line, err := json.Marshal(struct {
			Test    string  `json:"test"`
			Impl    string  `json:"impl"`
			Pass    bool    `json:"pass"`
			Metrics metrics `json:"metrics"`
		}{t.Name(), im.Name(), !t.Failed(), m})
		if err != nil {
			t.Error(err)
			return
		}
		f, err := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			t.Error(err)
			return
		}
		defer f.Close()
		fmt.Fprintf(f, "%s\n", line)
	})
}

// budget fails t when got exceeds limit, but only for implementations that
// carry delta d: v0.2.0 is measured, never judged, by recon's budgets.
func budget(t *testing.T, im impl.Impl, d workload.Delta, what string, got, limit float64) {
	t.Helper()
	if workload.Fixed(im, d) && got > limit {
		t.Errorf("%s = %.4g, budget %.4g", what, got, limit)
	}
}

// memSnap is the process memory that a scenario is charged for.
type memSnap struct {
	heap, stack, mallocs uint64
	goroutines           int
}

// snapshot waits for the goroutine count to settle (v0.2's per-call
// watcher can outlive its Do by a moment, holding the call's context),
// collects twice, then reads the live heap, stacks, cumulative allocations
// and goroutine count.
func snapshot() memSnap {
	workload.SettledGoroutines()
	runtime.GC()
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return memSnap{ms.HeapAlloc, ms.StackInuse, ms.Mallocs, runtime.NumGoroutine()}
}

// per is (after - before) / n for each field, as signed floats.
func (a memSnap) per(before memSnap, n int) (heap, stack, mallocs, goroutines float64) {
	d := func(x, y uint64) float64 { return (float64(x) - float64(y)) / float64(n) }
	return d(a.heap, before.heap), d(a.stack, before.stack), d(a.mallocs, before.mallocs),
		float64(a.goroutines-before.goroutines) / float64(n)
}

// peaks samples goroutines (beyond those the scenario accounts for) and
// the in-use heap every millisecond until stop is called.
type peaks struct {
	extraG, heap atomic.Int64
	done         chan struct{}
	wg           sync.WaitGroup
}

// watch starts sampling; accounted reports the goroutines the scenario
// itself runs (base plus its callers) at the moment of sampling.
func watch(accounted func() int64) *peaks {
	p := &peaks{done: make(chan struct{})}
	p.wg.Go(func() {
		tick := time.NewTicker(time.Millisecond)
		defer tick.Stop()
		var ms runtime.MemStats
		for i := 0; ; i++ {
			select {
			case <-p.done:
				return
			case <-tick.C:
			}
			raise(&p.extraG, int64(runtime.NumGoroutine())-accounted())
			if i%10 == 0 { // ReadMemStats stops the world; keep it rare
				runtime.ReadMemStats(&ms)
				raise(&p.heap, int64(ms.HeapInuse))
			}
		}
	})
	return p
}

func (p *peaks) stop() (extraG, heapMB float64) {
	close(p.done)
	p.wg.Wait()
	return float64(p.extraG.Load()), float64(p.heap.Load()) / (1 << 20)
}

func raise(a *atomic.Int64, v int64) {
	for {
		cur := a.Load()
		if v <= cur || a.CompareAndSwap(cur, v) {
			return
		}
	}
}

// waitFor polls cond every 100µs, failing t after d.
func waitFor(t *testing.T, d time.Duration, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(d); !cond(); time.Sleep(100 * time.Microsecond) {
		if time.Now().After(deadline) {
			t.Fatalf("%s: not reached within %v", what, d)
		}
	}
}

func ms(d time.Duration) float64 { return float64(d) / 1e6 }
