package industrial_test

import (
	"context"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// Spike shape: a budget of 8 units held ~1ms per call serves ~7000 calls/s.
// Base traffic uses under a third of it; the burst is 10× base, about three
// times what the budget can serve. Every caller has a 100ms deadline.
const (
	spikeCapacity = 8
	spikeService  = time.Millisecond
	spikeBase     = 2000.0 // arrivals per second
	spikeFactor   = 10
	spikeDeadline = 100 * time.Millisecond
)

// TestSpike holds open-loop traffic at the base rate, bursts it to 10×,
// then returns to base. Under the burst, callers whose deadline expires in
// the queue must leave holding nothing, so the budget keeps serving at its
// capacity; capacity is never exceeded; the burst's backlog must clear
// within a second of the burst's end, leaving the whole budget free.
// Reported latency counts from each call's scheduled arrival.
//
// The backlog, not the latency, is judged. Latency recovery (recovery_ms)
// is reported but measures the machine as much as the runtime: across 80
// runs per implementation, with other desktop software running, time.Sleep
// and scheduling stalls of 5-80ms put windows of the post phase over the
// pre-spike limit for both implementations alike (1 run in 4 under load,
// 1 in 30 when idle), long after every burst arrival had returned. The
// backlog cleared within one deadline (≤ 110ms) in every one of those runs.
func TestSpike(t *testing.T) {
	pre, burst, post := workload.Scale(time.Second, 5*time.Second), workload.Scale(500*time.Millisecond, 3*time.Second),
		workload.Scale(1500*time.Millisecond, 7*time.Second)
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		s := runSpike(t, im, pre, burst, post)
		record(t, im, s)
		if s["backlog_cleared_ms"] > 1000 {
			t.Errorf("the burst's last caller returned %.0fms after the burst", s["backlog_cleared_ms"])
		}
		// Callers and their deadline timers are counted exactly, so whatever
		// remains was started by the runtime under test (Δ10: none).
		budget(t, im, workload.D10, "goroutines created per call", s["goroutines_per_call"], 0.001)
	})
}

// call is one arrival's fate.
type call struct {
	at  time.Duration // scheduled arrival, since start
	lat time.Duration // scheduled arrival → return
	in  time.Duration // inside Do: call → return
	ok  bool
}

func runSpike(t *testing.T, im impl.Impl, pre, burst, post time.Duration) metrics {
	t.Helper()
	runtime.GC() // the first collection starts the GC workers; not this test's goroutines
	baseG := workload.SettledGoroutines()
	spec := impl.Spec{Name: "db", Capacity: spikeCapacity}
	rt, err := im.NewRuntime(spec)
	if err != nil {
		t.Fatal(err)
	}
	mon := workload.NewMonitor(spec)
	reqs := []impl.Req{{Name: "db", Units: 1}}
	op, err := rt.NewOp("write", impl.Policy{Effect: impl.Idempotent, Resources: reqs}, func(ctx context.Context, in int) (int, error) {
		defer mon.Enter(reqs, "")()
		time.Sleep(spikeService)
		return in, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	burstAt, burstEnd, end := pre, pre+burst, pre+burst+post
	rate := func(at time.Duration) float64 {
		if at >= burstAt && at < burstEnd {
			return spikeBase * spikeFactor
		}
		return spikeBase
	}
	var (
		mu       sync.Mutex
		calls    []call
		inflight atomic.Int64
		fired    atomic.Uint64 // callers' deadline timers that fired
		wg       sync.WaitGroup
	)
	pk := watch(func() int64 { return int64(baseG) + 1 + inflight.Load() }) // +1: the sampler
	createdBefore := created()
	start := workload.Now()
	r := workload.Rand(workload.Seed(t))
	for at := time.Duration(0); at < end; at += time.Duration(r.ExpFloat64() / rate(at) * float64(time.Second)) {
		if wait := at - workload.Since(start); wait > 0 {
			time.Sleep(wait) // ~0.5ms ticks on Windows: due arrivals leave together
		}
		inflight.Add(1)
		wg.Go(func() {
			defer inflight.Add(-1)
			called := workload.Now()
			// The caller's deadline is the test's own timer, so every fire, and
			// the goroutine Go runs it on, is counted exactly.
			ctx, cancel := context.WithCancelCause(context.Background())
			left := spikeDeadline - (workload.Since(start) - at)
			deadline := time.AfterFunc(max(left, 0), func() { fired.Add(1); cancel(context.DeadlineExceeded) })
			_, err := op.Do(ctx, 0)
			deadline.Stop()
			cancel(nil)
			c := call{at: at, lat: workload.Since(start) - at, in: workload.Since(called), ok: err == nil}
			mu.Lock()
			calls = append(calls, c)
			mu.Unlock()
		})
	}
	wg.Wait()
	workload.SettledGoroutines() // every fired timer has run and been counted
	spawned := created() - createdBefore
	extraG, peakHeap := pk.stop()
	wholeBudgetFree(t, rt)
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
	mon.Check(t)
	workload.NoLeak(t, baseG, 0, 2*time.Second)

	slices.SortFunc(calls, func(a, b call) int { return int(a.at - b.at) })
	phase := func(from, to time.Duration) []call {
		lo, _ := slices.BinarySearchFunc(calls, from, func(c call, at time.Duration) int { return int(c.at - at) })
		hi, _ := slices.BinarySearchFunc(calls, to, func(c call, at time.Duration) int { return int(c.at - at) })
		return calls[lo:hi]
	}
	settled := phase(pre/5, burstAt) // skip the first fifth: warm-up
	during := phase(burstAt, burstEnd)
	tail := phase(burstEnd+post/2, end)
	preP99 := quantile(settled, 0.99, inDo)
	limit := max(2*preP99, preP99+2*time.Millisecond)
	return metrics{
		"pre_p50_ms": ms(quantile(settled, 0.5, byArrival)), "pre_p99_ms": ms(quantile(settled, 0.99, byArrival)),
		"burst_p50_ms": ms(quantile(during, 0.5, byArrival)), "burst_p99_ms": ms(quantile(during, 0.99, byArrival)),
		"post_p99_ms":           ms(quantile(tail, 0.99, byArrival)),
		"pre_in_do_p99_ms":      ms(preP99),
		"launch_lag_p99_ms":     ms(quantile(calls, 0.99, func(c call) time.Duration { return c.lat - c.in })),
		"burst_goodput_per_s":   float64(okCount(during)) / burst.Seconds(),
		"burst_shed_frac":       1 - float64(okCount(during))/float64(max(len(during), 1)),
		"recovery_ms":           ms(recovery(phase(burstEnd, end), burstEnd, limit)),
		"backlog_cleared_ms":    ms(lastReturn(during) - burstEnd),
		"recovery_limit_ms":     ms(limit),
		"goroutines_per_call":   (float64(spawned) - float64(len(calls)) - float64(fired.Load())) / float64(len(calls)),
		"peak_extra_goroutines": extraG, "peak_heap_MB": peakHeap, "arrivals": float64(len(calls)),
	}
}

func byArrival(c call) time.Duration { return c.lat }
func inDo(c call) time.Duration      { return c.in }

// wholeBudgetFree fails t unless one call claiming every unit of the
// budget is admitted within a second: the burst's expired waiters must
// have left no units behind.
func wholeBudgetFree(t *testing.T, rt impl.Runtime) {
	t.Helper()
	reqs := []impl.Req{{Name: "db", Units: spikeCapacity}}
	whole, err := rt.NewOp("whole", impl.Policy{Effect: impl.Idempotent, Resources: reqs},
		func(_ context.Context, in int) (int, error) { return in, nil })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := whole.Do(ctx, 0); err != nil {
		t.Errorf("after the burst, a call claiming all %d units failed: %v", spikeCapacity, err)
	}
}

// quantile is the q-quantile of the successful calls' latency, as read by
// lat (0 if none succeeded).
func quantile(cs []call, q float64, lat func(call) time.Duration) time.Duration {
	var lats []time.Duration
	for _, c := range cs {
		if c.ok {
			lats = append(lats, lat(c))
		}
	}
	if len(lats) == 0 {
		return 0
	}
	slices.Sort(lats)
	return lats[min(len(lats)-1, int(q*float64(len(lats))))]
}

// lastReturn is when the last of cs returned, since start.
func lastReturn(cs []call) (last time.Duration) {
	for _, c := range cs {
		last = max(last, c.at+c.lat)
	}
	return last
}

func okCount(cs []call) (n int) {
	for _, c := range cs {
		if c.ok {
			n++
		}
	}
	return n
}

// recovery is how long after from the last 50ms window of arrivals ended
// whose successful p90 time inside Do exceeded limit, or whose calls mostly
// failed.
//
// Two choices keep the measure on the runtime. Judged by p99, a window of
// ~100 arrivals is judged by its slowest call or two (one timer hiccup made
// a 1100ms "recovery"). Judged from scheduled arrival, a late generator
// reads as a slow runtime (slow calls that started 5.9ms late spent 2.5ms in
// Do). The arrival process is open-loop, one goroutine per arrival, so time
// inside Do cannot hide a stalled runtime.
func recovery(cs []call, from, limit time.Duration) time.Duration {
	const window = 50 * time.Millisecond
	last := time.Duration(0)
	for i := 0; i < len(cs); {
		w := (cs[i].at - from) / window
		j := i
		for j < len(cs) && (cs[j].at-from)/window == w {
			j++
		}
		if win := cs[i:j]; quantile(win, 0.9, inDo) > limit || 2*okCount(win) < len(win) {
			last = (w + 1) * window
		}
		i = j
	}
	return last
}
