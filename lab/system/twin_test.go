package system_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// The Mirai twin, modeled on mirai-backend's cmd/bridge wiring and its
// four consumers, with every duration scaled by 1/100 (render's 20s
// attempt timeout becomes 200ms, the mock provider's 250ms becomes 2.5ms).
//
//	render     renderer:1  NonIdempotent Reject  timeout 200ms  ×2 on Transient
//	events     db-write:1  Idempotent    Wait    timeout  30ms  ×3 on Transient
//	catalogue  db-write:1  Idempotent    Wait    timeout  50ms  ×3 on Transient, keyed by brand
//	handoff    db-write:1  Idempotent    Wait    timeout  30ms  ×3 on Transient
const (
	render = iota
	events
	catalogue
	handoff
)

// twinTraffic is an open-loop arrival rate (per second) and a caller
// deadline (the HTTP request budget), per operation.
var twinTraffic = [...]struct {
	rate     float64
	deadline time.Duration
}{
	render:    {1000, 300 * time.Millisecond},
	events:    {1000, 100 * time.Millisecond},
	catalogue: {300, 100 * time.Millisecond},
	handoff:   {300, 100 * time.Millisecond},
}

// TestMiraiTwin runs the twin under open-loop traffic, then drains it with
// traffic still arriving, and judges it by SLO-style properties:
//
//   - no capacity violation (renderer ≤ 8, db-write ≤ 4) and no brand
//     written by two catalogue calls at once;
//   - no unsafe retry: render is never re-invoked after an Uncertain or
//     non-retryable attempt;
//   - p99 admission wait of the db-write (Wait) operations within half of
//     their 100ms request budget (50ms), and render shedding below 10% of
//     its traffic;
//   - a clean drain: Shutdown returns nil within 1s, the Stats ledger
//     reconciles, and goroutines return to baseline.
//
// LAB_LEVEL=full runs it as a 5-minute soak and additionally asserts that
// live heap and goroutines reach a steady state (last quarter of the run
// no higher than the first, beyond slack).
func TestMiraiTwin(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		seed := workload.Seed(t)
		baseG := workload.SettledGoroutines()
		specs := []impl.Spec{{Name: "renderer", Capacity: 8}, {Name: "db-write", Capacity: 4}}
		rt, err := im.NewRuntime(specs...)
		if err != nil {
			t.Fatal(err)
		}
		ops := twinOps()
		tr := newTracer(specs, ops)
		handles := make([]impl.Op, len(ops))
		for i, o := range ops {
			if handles[i], err = rt.NewOp(o.name, o.policy, twinHandler(tr, im, i, seed)); err != nil {
				t.Fatal(err)
			}
		}

		duration, every := workload.Scale(3*time.Second, 5*time.Minute), workload.Scale(500*time.Millisecond, 5*time.Second)
		stop := make(chan struct{})
		var requests, generators sync.WaitGroup
		for op := range ops {
			generators.Go(func() {
				arrivals(workload.Rand(seed+uint64(op)), twinTraffic[op].rate, stop, func() {
					requests.Go(func() {
						ctx, cancel := context.WithTimeout(context.Background(), twinTraffic[op].deadline)
						defer cancel()
						id := tr.issue(op)
						class, caught := call(im, handles[op], ctx, id)
						tr.finish(id, class, caught)
					})
				})
			})
		}

		var heap, gor []float64
		for deadline := time.Now().Add(duration); time.Now().Before(deadline); time.Sleep(every) {
			heap = append(heap, float64(workload.LiveHeap()))
			gor = append(gor, float64(runtime.NumGoroutine()))
		}
		var drainErr error
		drainTook := time.Now()
		workload.Within(t, 5*time.Second, "twin drain", func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			drainErr = rt.Shutdown(ctx) // traffic is still arriving
			if n := tr.running.Load(); drainErr == nil && n != 0 {
				drainErr = fmt.Errorf("Shutdown returned nil with %d handlers running", n)
			}
		})
		took := time.Since(drainTook)
		close(stop)
		generators.Wait()
		workload.Within(t, 5*time.Second, "requests after drain", requests.Wait)

		if drainErr != nil {
			t.Errorf("Shutdown under traffic: %v after %v", drainErr, took)
		}
		tr.check(t, im, rt.Stats())
		workload.NoLeak(t, baseG, 0, 2*time.Second)

		dbWait, renderWait := tr.admissionWaits(events, catalogue, handoff), tr.admissionWaits(render)
		tr.mu.Lock()
		l := tr.ledger
		tr.mu.Unlock()
		shed := float64(l.overloads) / float64(l.calls) // only render sheds
		t.Logf("%s; stats %+v", tr.summary(), rt.Stats())
		t.Logf("db-write admission wait: %v", dbWait)
		t.Logf("render admission wait:   %v; overloads %.2f%% of all calls; drain took %v", renderWait, 100*shed, took)
		if dbWait.P99 > twinTraffic[events].deadline/2 {
			t.Errorf("db-write p99 admission wait %v, SLO %v", dbWait.P99, twinTraffic[events].deadline/2)
		}
		if renderShare := twinTraffic[render].rate / totalRate(); shed > 0.10*renderShare {
			t.Errorf("render shed %.1f%% of its traffic, SLO 10%%", 100*shed/renderShare)
		}
		if workload.Full() {
			steady(t, "live heap (B)", heap, 4<<20)
			steady(t, "goroutines", gor, 64)
		}
		t.Logf("heap samples (MB): %v", mb(heap))
	})
}

func twinOps() []opSpec {
	on := map[impl.Outcome]bool{impl.Transient: true}
	db := []impl.Req{{Name: "db-write", Units: 1}}
	brand := func(in int) string { return "brand-" + strconv.Itoa(brandOf(in)) }
	return []opSpec{
		render: {name: "render", policy: impl.Policy{Effect: impl.NonIdempotent, Admission: impl.Reject,
			Resources: []impl.Req{{Name: "renderer", Units: 1}}, Timeout: 200 * time.Millisecond, MaxAttempts: 2, On: on}},
		events:    {name: "events", policy: impl.Policy{Effect: impl.Idempotent, Resources: db, Timeout: 30 * time.Millisecond, MaxAttempts: 3, On: on}},
		catalogue: {name: "catalogue", key: brand, policy: impl.Policy{Effect: impl.Idempotent, Resources: db, Timeout: 50 * time.Millisecond, MaxAttempts: 3, On: on, Key: brand}},
		handoff:   {name: "handoff", policy: impl.Policy{Effect: impl.Idempotent, Resources: db, Timeout: 30 * time.Millisecond, MaxAttempts: 3, On: on}},
	}
}

// dbService is the mean Postgres write time per db-write consumer.
var dbService = [...]time.Duration{events: 500 * time.Microsecond, catalogue: time.Millisecond, handoff: 700 * time.Microsecond}

// brandOf skews catalogue writes: brand 0 takes 40% of them.
func brandOf(in int) int {
	if in%5 < 2 {
		return 0
	}
	return 1 + in%9
}

// twinHandler plays the consumer's adapter: render is a provider that
// usually succeeds, sometimes fails before contacting the provider
// (Transient), rejects the input (Permanent), throttles, or hangs — and,
// as the v0.2 review recommends, maps its own timeout to Uncertain. The
// db-write consumers are Postgres writes with occasional serialization
// failures (Transient) and constraint violations (Permanent).
func twinHandler(tr *tracer, im impl.Impl, op int, seed uint64) impl.Handler {
	return func(ctx context.Context, in int) (int, error) {
		attempt, exit := tr.enter(in)
		returned, normal := impl.Permanent, false
		defer func() { exit(returned, normal) }()
		r := rand.New(rand.NewPCG(seed^uint64(in), uint64(attempt)))
		n := r.IntN(1000)
		var err error
		if op == render {
			switch {
			case n < 2: // hangs until its attempt deadline
				<-ctx.Done()
				returned, err = impl.Uncertain, im.Fail(impl.Uncertain, ctx.Err())
			case n < 42:
				returned, err = impl.Transient, im.Fail(impl.Transient, errAdapter)
			case n < 62:
				err = im.Fail(impl.Permanent, errAdapter)
			case n < 72:
				returned, err = impl.Throttled, im.Fail(impl.Throttled, errAdapter)
			default:
				time.Sleep(2*time.Millisecond + time.Duration(r.IntN(1000))*time.Microsecond)
			}
		} else {
			service := dbService[op]
			time.Sleep(service/2 + time.Duration(r.Int64N(int64(service))))
			switch {
			case n < 20:
				returned, err = impl.Transient, im.Fail(impl.Transient, errAdapter)
			case n < 30:
				err = im.Fail(impl.Permanent, errAdapter)
			}
		}
		if err == nil {
			returned = impl.Success
		}
		normal = true
		return in, err
	}
}

// arrivals calls emit as a Poisson process of the given rate until stop
// is closed. Arrivals that fall due while the generator sleeps (the timer
// tick is ~0.5ms on Windows) are emitted together, as bursts.
func arrivals(r *rand.Rand, rate float64, stop <-chan struct{}, emit func()) {
	next := time.Now()
	for {
		select {
		case <-stop:
			return
		default:
		}
		for now := time.Now(); !next.After(now); next = next.Add(time.Duration(r.ExpFloat64() / rate * float64(time.Second))) {
			emit()
		}
		time.Sleep(time.Until(next))
	}
}

func totalRate() float64 {
	sum := 0.0
	for _, tr := range twinTraffic {
		sum += tr.rate
	}
	return sum
}

// steady asserts the mean of the last quarter of samples is no higher than
// the mean of the first quarter after warm-up, plus slack.
func steady(t *testing.T, what string, xs []float64, slack float64) {
	t.Helper()
	q := len(xs) / 4
	if q < 2 {
		t.Errorf("%s: %d samples are too few for a steady-state check", what, len(xs))
		return
	}
	first, last := mean(xs[q:2*q]), mean(xs[len(xs)-q:])
	t.Logf("%s: early mean %.0f, late mean %.0f", what, first, last)
	if last > first+slack {
		t.Errorf("%s grows: early mean %.0f, late mean %.0f (slack %.0f)", what, first, last, slack)
	}
}

func mean(xs []float64) float64 {
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func mb(xs []float64) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = strconv.FormatFloat(x/(1<<20), 'f', 1, 64)
	}
	return out
}
