package assume

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// nsPerOp times n calls of f.
func nsPerOp(n int, f func()) float64 {
	start := time.Now()
	for range n {
		f()
	}
	return float64(time.Since(start).Nanoseconds()) / float64(n)
}

func sample(n int, f func() time.Duration) workload.Dist {
	ds := make([]time.Duration, n)
	for i := range ds {
		ds[i] = f()
	}
	return workload.Summarize(ds)
}

// TestClockResolution
//
// Assumption: the wall clock resolves single calls, so a call's latency
// can be read as time.Since(start).
//
// Why it matters: every per-call latency in the lab and the DoE (p99 Do,
// admission wait, wake-up latency) is a difference of two clock reads.
//
// Verdict: on Windows time.Now advances in steps of about 0.5ms, so the
// assumption is false there and the lab samples latency with
// workload.Now (the performance counter). Asserts the lab clock resolves
// below 1µs; logs both step sizes and the lab clock's read cost.
func TestClockResolution(t *testing.T) {
	stdStep := medianStep(func() int64 { return time.Now().UnixNano() })
	labStep := medianStep(func() int64 { return int64(workload.Now()) })
	cost := nsPerOp(200000, func() { workload.Now() })
	t.Logf("time.Now step %v; lab clock step %v, %.0f ns per read", stdStep, labStep, cost)
	if labStep >= time.Microsecond {
		t.Errorf("lab clock step %v: cannot time a single call", labStep)
	}
}

// medianStep is the median increment seen when polling now until it moves.
func medianStep(now func() int64) time.Duration {
	steps := make([]time.Duration, 21)
	for i := range steps {
		t0 := now()
		t1 := t0
		for t1 == t0 {
			t1 = now()
		}
		steps[i] = time.Duration(t1 - t0)
	}
	return workload.Summarize(steps).P50
}

// TestTimerResolution
//
// Assumption: timers honor millisecond durations (Go ≥1.23 uses
// high-resolution waitable timers on Windows; older runtimes rounded up to
// the 15.6ms system tick).
//
// Why it matters: per-attempt Timeouts of a few ms, and every real-time
// lab scenario, are meaningless if the platform rounds them to 15ms.
//
// Verdict: asserts Sleep(1ms) and a 1ms context deadline have p50 < 3ms.
// Logs the distributions: on Windows anything below the ~0.5ms timer tick
// is rounded up to it (Sleep(100µs) takes ~0.55ms), so sub-millisecond
// Timeouts do not mean what they say there.
func TestTimerResolution(t *testing.T) {
	for _, d := range []time.Duration{100 * time.Microsecond, time.Millisecond} {
		dist := sample(200, func() time.Duration {
			s := workload.Now()
			time.Sleep(d)
			return workload.Since(s)
		})
		t.Logf("Sleep(%v): %v", d, dist)
		if d == time.Millisecond && dist.P50 > 3*time.Millisecond {
			t.Errorf("Sleep(1ms) p50 = %v: coarse timers", dist.P50)
		}
	}
	dist := sample(200, func() time.Duration {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		s := workload.Now()
		<-ctx.Done()
		return workload.Since(s)
	})
	t.Logf("WithTimeout(1ms) fires after: %v", dist)
	if dist.P50 > 3*time.Millisecond {
		t.Errorf("1ms context deadline p50 = %v: coarse timers", dist.P50)
	}
}

// TestContextCosts
//
// Assumption: context.AfterFunc registration costs at most 2 allocations,
// and a bare context.WithTimeout costs a small fixed number.
//
// Why it matters: CONTRACT §10 prices a blocked waiter at ≤ 2 allocs
// (only possible if the wake-up hook is AfterFunc-cheap) and the Timeout>0
// path at "a bare WithTimeout + 0". These numbers are the budgets' anchors.
//
// Verdict: asserts AfterFunc+stop ≤ 2 allocs; logs allocs and ns for
// WithCancel, WithTimeout, AfterFunc, and AfterFunc's wake-up latency.
func TestContextCosts(t *testing.T) {
	bg := context.Background()
	parent, stopParent := context.WithCancel(bg)
	defer stopParent()
	cases := []struct {
		name string
		f    func()
	}{
		{"WithCancel+cancel", func() { _, c := context.WithCancel(bg); c() }},
		{"WithTimeout+cancel", func() { _, c := context.WithTimeout(bg, time.Hour); c() }},
		{"AfterFunc+stop", func() { stop := context.AfterFunc(parent, func() {}); stop() }},
	}
	for _, c := range cases {
		allocs := testing.AllocsPerRun(2000, c.f)
		t.Logf("%-20s %4.1f allocs  %6.0f ns", c.name, allocs, nsPerOp(200000, c.f))
		if c.name == "AfterFunc+stop" && allocs > 2 {
			t.Errorf("AfterFunc+stop costs %.1f allocs; the §10 waiter budget (≤2) is unreachable", allocs)
		}
	}
	wake := sample(500, func() time.Duration {
		ctx, cancel := context.WithCancel(bg)
		fired := make(chan workload.Stamp, 1)
		context.AfterFunc(ctx, func() { fired <- workload.Now() })
		s := workload.Now()
		cancel()
		return time.Duration(<-fired - s)
	})
	t.Logf("AfterFunc cancel→callback latency: %v", wake)
}

// TestGoroutineCost
//
// Assumption: starting a goroutine per call is expensive relative to the
// admission work itself (an uncontended lock).
//
// Why it matters: v0.2 starts a watcher goroutine per Do (D10); recon's
// "0 goroutines per Do" budget is only worth having if this cost is real.
//
// Verdict: asserts go+join costs more than 5× a mutex Lock/Unlock; logs both.
func TestGoroutineCost(t *testing.T) {
	var wg sync.WaitGroup
	spawn := nsPerOp(100000, func() {
		wg.Add(1)
		go wg.Done()
		wg.Wait()
	})
	var mu sync.Mutex
	lock := nsPerOp(1000000, func() { mu.Lock(); mu.Unlock() })
	allocs := testing.AllocsPerRun(1000, func() { wg.Add(1); go wg.Done(); wg.Wait() })
	t.Logf("go+join %.0f ns (%.1f allocs); mutex Lock/Unlock %.1f ns; ratio %.0f×", spawn, allocs, lock, spawn/lock)
	if spawn < 5*lock {
		t.Errorf("goroutine start is only %.1f× a mutex round trip", spawn/lock)
	}
}

// TestAtomicVsLocks
//
// Assumption: one atomic word is cheaper than v0.2's RWMutex read lock for
// the admission gate, uncontended and under parallel load.
//
// Why it matters: CONTRACT §9 requires lock-free admission/drain state;
// that choice is only justified if the atomic path is actually cheaper.
//
// Verdict: asserts the uncontended atomic Add beats RLock+RUnlock; logs
// uncontended and 20-way contended costs of atomic, Mutex and RWMutex.
// raceEnabled is set by race_test.go under -race.
var raceEnabled bool

func TestAtomicVsLocks(t *testing.T) {
	var n atomic.Int64
	var mu sync.Mutex
	var rw sync.RWMutex
	ops := []struct {
		name string
		f    func()
	}{
		{"atomic.Add", func() { n.Add(1) }},
		{"Mutex", func() { mu.Lock(); mu.Unlock() }},
		{"RWMutex.RLock", func() { rw.RLock(); rw.RUnlock() }},
	}
	solo := map[string]float64{}
	for _, op := range ops {
		solo[op.name] = nsPerOp(2000000, op.f)
		par := parallelNs(runtime.GOMAXPROCS(0), 200000, op.f)
		t.Logf("%-14s uncontended %5.1f ns   %d-way contended %6.1f ns/op", op.name, solo[op.name], runtime.GOMAXPROCS(0), par)
	}
	if raceEnabled {
		return // the race detector instruments atomics and locks unequally
	}
	if solo["atomic.Add"] >= solo["RWMutex.RLock"] {
		t.Errorf("atomic.Add (%.1f ns) is not cheaper than RLock+RUnlock (%.1f ns)", solo["atomic.Add"], solo["RWMutex.RLock"])
	}
}

func parallelNs(workers, each int, f func()) float64 {
	var wg sync.WaitGroup
	start := time.Now()
	for range workers {
		wg.Go(func() {
			for range each {
				f()
			}
		})
	}
	wg.Wait()
	return float64(time.Since(start).Nanoseconds()) / float64(each)
}

// TestGoroutineSettling
//
// Assumption: runtime.NumGoroutine drops back to its baseline promptly
// once goroutines return.
//
// Why it matters: every leak check in the lab (and the D8/D10 ledger)
// compares goroutine counts; if exits are reaped lazily, a correct
// implementation would look leaky.
//
// Verdict: asserts 10 000 exited goroutines are reaped within 1s; logs how
// long it took.
func TestGoroutineSettling(t *testing.T) {
	base := runtime.NumGoroutine()
	var wg sync.WaitGroup
	release := make(chan struct{})
	for range 10000 {
		wg.Go(func() { <-release })
	}
	close(release)
	wg.Wait()
	start := time.Now()
	n := workload.Goroutines(base, time.Second)
	t.Logf("10000 goroutines reaped to %d (baseline %d) in %v", n, base, time.Since(start))
	if n > base {
		t.Errorf("%d goroutines still counted 1s after they returned", n-base)
	}
}

// TestGCTailLatency
//
// Assumption: garbage collection does not dominate the tail latency of an
// uncontended Do.
//
// Why it matters: a per-call allocation budget (§10) is a tail-latency
// budget only if GC pauses are what the allocations cost; this measures
// whether GC pressure from elsewhere in the process moves Herta's p99.
//
// Verdict: asserts p99 under heavy GC pressure stays below 10ms; logs the
// distributions with and without pressure, per implementation.
func TestGCTailLatency(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 1})
		if err != nil {
			t.Fatal(err)
		}
		defer rt.Shutdown(context.Background())
		op, err := rt.NewOp("noop", impl.Policy{Effect: impl.Pure, Resources: []impl.Req{{Name: "r", Units: 1}}},
			func(ctx context.Context, in int) (int, error) { return in, nil })
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		measure := func() workload.Dist {
			return sample(workload.Scale(20000, 500000), func() time.Duration {
				s := workload.Now()
				op.Do(ctx, 1)
				return workload.Since(s)
			})
		}
		calm := measure()
		stop := make(chan struct{})
		var wg sync.WaitGroup
		wg.Go(func() {
			var keep [][]byte
			for {
				select {
				case <-stop:
					return
				default:
				}
				keep = append(keep, make([]byte, 64<<10))
				if len(keep) > 256 {
					keep = keep[:0]
					runtime.GC()
				}
			}
		})
		pressured := measure()
		close(stop)
		wg.Wait()
		t.Logf("calm:        %v", calm)
		t.Logf("GC pressure: %v", pressured)
		if pressured.P99 > 10*time.Millisecond {
			t.Errorf("p99 under GC pressure is %v", pressured.P99)
		}
	})
}
