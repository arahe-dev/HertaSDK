package industrial_test

import (
	"context"
	"errors"
	"math/rand/v2"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/doe"
	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// soakOps is the Mirai twin of lab/system (TestMiraiTwin): mirai-backend's
// four consumers with every duration scaled by 1/100, under the same
// open-loop rates and caller deadlines. render has the renderer budget (8)
// to itself and sheds; the other three share db-write (4) and queue.
var soakOps = []struct {
	name     string
	rate     float64       // arrivals per second
	deadline time.Duration // the caller's request budget
	service  time.Duration // mean handler time
	policy   impl.Policy
}{
	{"render", 1000, 300 * time.Millisecond, 2500 * time.Microsecond, impl.Policy{Effect: impl.NonIdempotent,
		Admission: impl.Reject, Resources: []impl.Req{{Name: "renderer", Units: 1}}, Timeout: 200 * time.Millisecond,
		MaxAttempts: 2, On: map[impl.Outcome]bool{impl.Transient: true}}},
	{"events", 1000, 100 * time.Millisecond, 500 * time.Microsecond, dbPolicy(30*time.Millisecond, nil)},
	{"catalogue", 300, 100 * time.Millisecond, time.Millisecond, dbPolicy(50*time.Millisecond, brand)},
	{"handoff", 300, 100 * time.Millisecond, 700 * time.Microsecond, dbPolicy(30*time.Millisecond, nil)},
}

var errSoak = errors.New("soak: consumer failure")

var soakSpecs = []impl.Spec{{Name: "renderer", Capacity: 8}, {Name: "db-write", Capacity: 4}}

func dbPolicy(timeout time.Duration, key func(int) string) impl.Policy {
	return impl.Policy{Effect: impl.Idempotent, Resources: []impl.Req{{Name: "db-write", Units: 1}},
		Timeout: timeout, MaxAttempts: 3, On: map[impl.Outcome]bool{impl.Transient: true}, Key: key}
}

// brand skews catalogue writes: brand 0 takes 40% of them.
func brand(in int) string {
	if in%5 < 2 {
		return "brand-0"
	}
	return "brand-" + strconv.Itoa(1+in%9)
}

// TestSoak holds the Mirai twin for LAB_SOAK (default 3s) and splits the run
// into ten windows. Per window it records the p99 time inside Do of the
// db-write consumers and of render, the live heap and the goroutine count.
// It then drains under traffic. Judged on every run: no capacity or key
// violation, a nil Shutdown, Finished == Admitted, no leaked goroutine.
// Judged on a soak of a minute or more: the last three windows' live heap,
// goroutines and db-write p99 are no higher than the first three's (after
// the first window, the warm-up) beyond slack.
func TestSoak(t *testing.T) {
	dur := 3 * time.Second
	if s := os.Getenv("LAB_SOAK"); s != "" {
		var err error
		if dur, err = time.ParseDuration(s); err != nil {
			t.Fatalf("LAB_SOAK: %v", err)
		}
	}
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		s := runSoak(t, im, dur)
		record(t, im, s.metrics())
		if dur < time.Minute {
			return
		}
		early, late := s.span(1, 4), s.span(7, 10)
		for _, c := range []struct {
			what        string
			early, late float64
			slack       float64
		}{
			{"live heap (MB)", early.heapMB, late.heapMB, 4},
			{"goroutines", early.goroutines, late.goroutines, 64},
			{"db-write p99 (ms)", early.dbP99, late.dbP99, early.dbP99 + 5},
		} {
			if c.late > c.early+c.slack {
				t.Errorf("%s drifts: windows 2-4 %.2f, windows 8-10 %.2f (slack %.2f)", c.what, c.early, c.late, c.slack)
			}
		}
	})
}

// soakWindow is one tenth of a soak.
type soakWindow struct {
	db, render doe.Hist // time inside Do, ns
	heapMB     float64
	goroutines float64
}

type soak struct {
	windows             []*soakWindow
	calls, ok, overload int
	drain               time.Duration
	heapPeakMB          float64
}

// span averages windows [from, to): heap and goroutines as means, the
// p99s from the merged histograms.
type spanStats struct{ heapMB, goroutines, dbP99, renderP99 float64 }

func (s soak) span(from, to int) spanStats {
	var db, render doe.Hist
	var out spanStats
	for _, w := range s.windows[from:to] {
		db.Merge(&w.db)
		render.Merge(&w.render)
		out.heapMB += w.heapMB / float64(to-from)
		out.goroutines += w.goroutines / float64(to-from)
	}
	out.dbP99, out.renderP99 = db.Quantile(0.99)/1e6, render.Quantile(0.99)/1e6
	return out
}

func (s soak) metrics() metrics {
	early, late := s.span(1, 4), s.span(7, 10)
	return metrics{
		"calls": float64(s.calls), "ok_frac": float64(s.ok) / float64(s.calls),
		"render_shed_frac": float64(s.overload) / float64(s.calls), "drain_ms": ms(s.drain),
		"heap_early_MB": early.heapMB, "heap_late_MB": late.heapMB, "heap_peak_MB": s.heapPeakMB,
		"goroutines_early": early.goroutines, "goroutines_late": late.goroutines,
		"db_p99_early_ms": early.dbP99, "db_p99_late_ms": late.dbP99,
		"render_p99_early_ms": early.renderP99, "render_p99_late_ms": late.renderP99,
		"db_p99_drift": late.dbP99 / early.dbP99,
	}
}

func runSoak(t *testing.T, im impl.Impl, dur time.Duration) soak {
	t.Helper()
	baseG := workload.SettledGoroutines()
	rt, err := im.NewRuntime(soakSpecs...)
	if err != nil {
		t.Fatal(err)
	}
	mon := workload.NewMonitor(soakSpecs...)
	ops := make([]impl.Op, len(soakOps))
	for i, o := range soakOps {
		if ops[i], err = rt.NewOp(o.name, o.policy, soakHandler(im, mon, i)); err != nil {
			t.Fatal(err)
		}
	}

	s := soak{windows: make([]*soakWindow, 10)}
	for i := range s.windows {
		s.windows[i] = new(soakWindow)
	}
	win := dur / 10
	var (
		mu       sync.Mutex
		requests sync.WaitGroup
		gens     sync.WaitGroup
	)
	stop := make(chan struct{})
	start := workload.Now()
	for i, o := range soakOps {
		gens.Go(func() {
			poisson(rand.New(rand.NewPCG(uint64(i), workload.Seed(t))), o.rate, stop, func() {
				requests.Go(func() {
					ctx, cancel := context.WithTimeout(context.Background(), o.deadline)
					defer cancel()
					at := workload.Now()
					_, err := ops[i].Do(ctx, int(at))
					in := workload.Since(at)
					w := s.windows[min(int(time.Duration(at-start)/win), len(s.windows)-1)]
					if i == 0 {
						w.render.Record(int64(in))
					} else {
						w.db.Record(int64(in))
					}
					mu.Lock()
					s.calls++
					switch im.Classify(err) {
					case impl.ClassNil:
						s.ok++
					case impl.ClassOverloaded:
						s.overload++
					}
					mu.Unlock()
				})
			})
		})
	}
	for i, w := range s.windows {
		if wait := time.Duration(i+1)*win - workload.Since(start); wait > 0 {
			time.Sleep(wait)
		}
		w.heapMB = float64(workload.LiveHeap()) / (1 << 20)
		w.goroutines = float64(runtime.NumGoroutine())
		s.heapPeakMB = max(s.heapPeakMB, w.heapMB)
	}

	drainAt := workload.Now() // traffic is still arriving
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := rt.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown under traffic: %v", err)
	}
	cancel()
	s.drain = workload.Since(drainAt)
	close(stop)
	gens.Wait()
	workload.Within(t, 5*time.Second, "requests after drain", requests.Wait)
	if st := rt.Stats(); st.Finished != st.Admitted {
		t.Errorf("after the drain Finished = %d, Admitted = %d", st.Finished, st.Admitted)
	}
	mon.Check(t)
	workload.NoLeak(t, baseG, 0, 2*time.Second)
	return s
}

// soakHandler plays the consumers: render usually succeeds after 2-3ms, and
// sometimes hangs to its attempt deadline (reported Uncertain), fails
// Transient, Permanent or Throttled; a db write takes 0.5-1.5× its mean
// service time and sometimes fails Transient (serialization) or Permanent.
func soakHandler(im impl.Impl, mon *workload.Monitor, op int) impl.Handler {
	o := soakOps[op]
	return func(ctx context.Context, in int) (int, error) {
		key := ""
		if o.policy.Key != nil {
			key = o.policy.Key(in)
		}
		defer mon.Enter(o.policy.Resources, key)()
		n := rand.IntN(1000)
		if op == 0 {
			switch {
			case n < 2:
				<-ctx.Done()
				return 0, im.Fail(impl.Uncertain, ctx.Err())
			case n < 42:
				return 0, im.Fail(impl.Transient, errSoak)
			case n < 62:
				return 0, im.Fail(impl.Permanent, errSoak)
			case n < 72:
				return 0, im.Fail(impl.Throttled, errSoak)
			}
		}
		time.Sleep(o.service/2 + rand.N(o.service))
		switch {
		case op == 0:
		case n < 20:
			return 0, im.Fail(impl.Transient, errSoak)
		case n < 30:
			return 0, im.Fail(impl.Permanent, errSoak)
		}
		return in, nil
	}
}

// poisson calls emit at the given mean rate until stop closes. Arrivals
// that fall due while the generator sleeps leave together.
func poisson(r *rand.Rand, rate float64, stop <-chan struct{}, emit func()) {
	next := workload.Now()
	for {
		select {
		case <-stop:
			return
		default:
		}
		for now := workload.Now(); next <= now; next += workload.Stamp(r.ExpFloat64() / rate * 1e9) {
			emit()
		}
		if wait := time.Duration(next - workload.Now()); wait > 0 {
			time.Sleep(wait)
		}
	}
}
