package system_test

import (
	"context"
	"errors"
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

// loadConfig is one concurrent workload.
type loadConfig struct {
	workers, perWorker int
	chaos              bool // panics, ctx-ignoring handlers, caller cancellations
	shutdownMidway     bool // Shutdown (by several callers) while load runs
}

// TestInvariantsUnderLoad: a random topology (3 budgets, 8 operations
// with random effects, admission, multi-resource claims, keys, timeouts
// and retry tables) under 64 concurrent callers. Every handler checks
// capacity, key exclusion and retry safety; after drain the Stats ledger
// must reconcile with the per-call trace, and goroutines and heap must
// return to their baselines.
func TestInvariantsUnderLoad(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rnd := workload.Rand(workload.Seed(t))
		for range workload.Scale(3, 30) {
			runLoad(t, im, rnd, loadConfig{workers: 64, perWorker: workload.Scale(150, 2000)})
		}
	})
}

// TestChaos adds what production adds: handlers that panic or ignore
// their context, callers that cancel or time out mid-flight, and Shutdown
// at a random moment from several goroutines at once (with short
// timeouts, retried until nil) while traffic continues. The same
// invariants must hold, and nothing may hang.
func TestChaos(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rnd := workload.Rand(workload.Seed(t))
		for range workload.Scale(5, 100) {
			runLoad(t, im, rnd, loadConfig{workers: 64, perWorker: workload.Scale(150, 2000), chaos: true, shutdownMidway: true})
		}
	})
}

var errAdapter = errors.New("lab: adapter failure")

func runLoad(t *testing.T, im impl.Impl, rnd *rand.Rand, cfg loadConfig) {
	t.Helper()
	baseG, baseHeap := workload.SettledGoroutines(), workload.LiveHeap()
	specs := []impl.Spec{{Name: "a", Capacity: 2}, {Name: "b", Capacity: 3}, {Name: "c", Capacity: 4}}
	rt, err := im.NewRuntime(specs...)
	if err != nil {
		t.Fatal(err)
	}
	ops := randomOps(rnd, specs, 8)
	tr := newTracer(specs, ops)
	var handles []impl.Op
	for i, o := range ops {
		h, err := rt.NewOp(o.name, o.policy, loadHandler(tr, im, rnd.Uint64(), cfg.chaos))
		if err != nil {
			t.Fatalf("op %d: %v", i, err)
		}
		handles = append(handles, h)
	}

	var drained sync.Once
	var drainErr error
	shutdown := func() {
		drained.Do(func() {
			var wg sync.WaitGroup
			for range 4 { // concurrent Shutdown callers, each retrying on timeout
				wg.Go(func() {
					for {
						ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rnd2().IntN(3000))*time.Microsecond)
						err := rt.Shutdown(ctx)
						cancel()
						if err == nil {
							if n := tr.running.Load(); n != 0 {
								drainErr = fmt.Errorf("Shutdown returned nil with %d handlers running", n)
							}
							return
						}
					}
				})
			}
			wg.Wait()
		})
	}

	seeds := make([]uint64, cfg.workers)
	for i := range seeds {
		seeds[i] = rnd.Uint64()
	}
	stopAt := time.Duration(50+rnd.IntN(1500)) * time.Millisecond
	workload.Within(t, 120*time.Second, "load round", func() {
		var wg sync.WaitGroup
		if cfg.shutdownMidway {
			wg.Go(func() { time.Sleep(stopAt); shutdown() })
		}
		for w := range cfg.workers {
			wg.Go(func() {
				wr := workload.Rand(seeds[w])
				for range cfg.perWorker {
					op := wr.IntN(len(ops))
					ctx, cancel := callerContext(wr, cfg.chaos)
					id := tr.issue(op)
					class, caught := call(im, handles[op], ctx, id)
					cancel()
					tr.finish(id, class, caught)
				}
			})
		}
		wg.Wait()
		shutdown()
	})
	if drainErr != nil {
		t.Error(drainErr)
	}
	tr.check(t, im, rt.Stats())
	workload.NoLeak(t, baseG, 0, 2*time.Second)
	if heap := workload.LiveHeap(); heap > baseHeap+4<<20 {
		t.Errorf("live heap %d B above baseline after drain", heap-baseHeap)
	}
	if t.Failed() {
		t.Fatalf("%s; stats %+v", tr.summary(), rt.Stats())
	}
	t.Logf("%s; stats %+v", tr.summary(), rt.Stats())
}

// rnd2 is a per-goroutine generator for incidental jitter.
func rnd2() *rand.Rand { return rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())) }

// call runs Do, converting a propagated handler panic into caught=true.
func call(im impl.Impl, op impl.Op, ctx context.Context, id int) (class impl.Class, caught bool) {
	defer func() {
		if recover() != nil {
			class, caught = impl.ClassOther, true
		}
	}()
	_, err := op.Do(ctx, id)
	return im.Classify(err), false
}

func callerContext(r *rand.Rand, chaos bool) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	if !chaos {
		return context.WithCancel(ctx)
	}
	switch n := r.IntN(10); {
	case n < 2:
		return context.WithTimeout(ctx, time.Duration(500+r.IntN(4500))*time.Microsecond)
	case n < 3:
		ctx, cancel := context.WithCancel(ctx)
		time.AfterFunc(time.Duration(r.IntN(3000))*time.Microsecond, cancel)
		return ctx, cancel
	}
	return context.WithCancel(ctx)
}

func randomOps(rnd *rand.Rand, specs []impl.Spec, n int) []opSpec {
	var ops []opSpec
	for i := range n {
		p := impl.Policy{Effect: impl.Effects[rnd.IntN(3)], MaxAttempts: 1 + rnd.IntN(3)}
		if rnd.IntN(4) == 0 {
			p.Admission = impl.Reject
		}
		if rnd.IntN(2) == 0 {
			p.Timeout = 2 * time.Millisecond
		}
		for _, j := range rnd.Perm(len(specs))[:1+rnd.IntN(2)] {
			p.Resources = append(p.Resources, impl.Req{Name: specs[j].Name, Units: 1 + rnd.Int64N(specs[j].Capacity)})
		}
		p.On = map[impl.Outcome]bool{impl.Transient: rnd.IntN(2) == 0, impl.Uncertain: p.Effect != impl.NonIdempotent && rnd.IntN(2) == 0}
		o := opSpec{name: fmt.Sprint("op", i), policy: p}
		if rnd.IntN(10) < 3 {
			o.key = func(in int) string { return "k" + strconv.Itoa(in%5) }
			o.policy.Key = o.key
		}
		ops = append(ops, o)
	}
	return ops
}

// loadHandler plays a seeded mix of outcomes; chaos adds panics and
// handlers that ignore their context.
func loadHandler(tr *tracer, im impl.Impl, seed uint64, chaos bool) impl.Handler {
	return func(ctx context.Context, in int) (int, error) {
		attempt, exit := tr.enter(in)
		returned, normal := impl.Permanent, false
		defer func() { exit(returned, normal) }()
		r := rand.New(rand.NewPCG(seed^uint64(in), uint64(attempt)))
		n := r.IntN(100)
		if !chaos {
			n = min(n, 93) // no panics, no ctx-ignoring sleeps
		}
		var err error
		switch {
		case n < 55:
			for range r.IntN(20) {
				runtime.Gosched()
			}
		case n < 62:
			time.Sleep(time.Millisecond)
		case n < 72:
			returned, err = impl.Transient, im.Fail(impl.Transient, errAdapter)
		case n < 77:
			returned, err = impl.Uncertain, im.Fail(impl.Uncertain, errAdapter)
		case n < 81:
			err = im.Fail(impl.Permanent, errAdapter)
		case n < 84:
			returned, err = impl.Throttled, im.Fail(impl.Throttled, errAdapter)
		case n < 88:
			err = errAdapter
		case n < 94: // a provider adapter that maps its own timeout to Transient
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Millisecond):
			}
			returned, err = impl.Transient, im.Fail(impl.Transient, errAdapter)
		case n < 97: // ignores ctx entirely
			time.Sleep(3 * time.Millisecond)
		default:
			panic("lab: chaos panic")
		}
		if err == nil {
			returned = impl.Success
		}
		normal = true
		return in, err
	}
}
