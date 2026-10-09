package assume

// The CONTRACT §8 ledger. Each test asserts v0.2's defect on orig and the
// fix on every later implementation, keyed by workload.Fixed. Pinned
// elsewhere: D2 and DNondet in lab/model (drain_test.go, as rates, because
// v0.2 shows them only under races). D5 has no test: lab/impl copies
// Policy.On into a fresh map at NewOp, so mutating the caller's map after
// construction cannot reach either implementation through the adapter.

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestD1RejectWithHeldKey: a keyed Reject call whose key is held blocks in
// v0.2; under the contract it fails fast with Fail(Throttled,
// ErrOverloaded) and counts in Rejected.
func TestD1RejectWithHeldKey(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		op := op1("keyed-reject", impl.Reject, 1)
		op.Keyed = true
		sc := model.Scenario{
			Specs: []impl.Spec{{Name: "r", Capacity: 4}},
			Ops:   []model.Op{op},
			Calls: calls(2, 0, "k", held),
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			start(r, 0, 1)
			if !workload.Fixed(im, workload.D1) {
				expect(t, r, model.Parked, 1)
				return
			}
			expectResult(t, r, 1, impl.ClassOverloaded)
			if res, st := r.Result(1), r.View().Stats; res.Outcome != impl.Throttled || st.Rejected != 1 {
				t.Errorf("result %v, stats %+v; want Throttled and Rejected 1", res, st)
			}
		})
	})
}

// TestD3PlainErrorAfterDeadline: a handler that outlives its attempt
// deadline and returns an untagged error is reported Uncertain by v0.2 but
// not retried; under the contract the same verdict drives the retry.
func TestD3PlainErrorAfterDeadline(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		op := model.Op{Effect: impl.Pure, Timeout: 10 * time.Millisecond, MaxAttempts: 3, On: []impl.Outcome{impl.Uncertain}}
		sc := solo(op, model.Call{Script: model.Script{{Block: model.BlockDeadline, Return: model.ReturnPlain}}})
		calls, elapsed := 1, 10*time.Millisecond
		if workload.Fixed(im, workload.D3) {
			calls, elapsed = 3, 30*time.Millisecond
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			r.Start(0)
			if res := r.Await(0); res.Outcome != impl.Uncertain || res.Calls != calls || r.Elapsed(0) != elapsed {
				t.Errorf("got %v in %v; want Uncertain after %d calls in %v", res, r.Elapsed(0), calls, elapsed)
			}
		})
	})
}

// TestD4CallerDeadline: when the caller's own deadline ends an attempt,
// v0.2 reports Uncertain or Permanent depending on Retry.On; under the
// contract it is always Permanent and never retried.
func TestD4CallerDeadline(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		noRetry := model.Op{Name: "no-retry", Effect: impl.Pure, Timeout: 10 * time.Millisecond}
		retry := model.Op{Name: "retry", Effect: impl.Pure, Timeout: 10 * time.Millisecond, MaxAttempts: 3, On: []impl.Outcome{impl.Uncertain}}
		wait := model.Script{{Block: model.BlockDeadline, Return: model.ReturnCtxErr}}
		sc := model.Scenario{
			Specs: []impl.Spec{{Name: "unused", Capacity: 1}},
			Ops:   []model.Op{noRetry, retry},
			Calls: []model.Call{{Op: 0, Deadline: 5500 * time.Microsecond, Script: wait}, {Op: 1, Deadline: 5500 * time.Microsecond, Script: wait}},
		}
		want := []impl.Outcome{impl.Uncertain, impl.Permanent}
		if workload.Fixed(im, workload.D4) {
			want = []impl.Outcome{impl.Permanent, impl.Permanent}
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			for i := range 2 {
				r.Start(i)
				if res := r.Await(i); res.Class != impl.ClassDeadline || res.Outcome != want[i] || res.Calls != 1 {
					t.Errorf("%s: got %v, want deadline/%v after 1 call", sc.Ops[i].Name, res, want[i])
				}
			}
		})
	})
}

// TestD6CallerErrorBeatsStopping: a waiter whose caller cancelled just
// before Shutdown must report the caller's ctx.Err(); v0.2 reports
// ErrRuntimeStopping whenever Shutdown has begun by the time it looks. The
// rate is measured.
func TestD6CallerErrorBeatsStopping(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		n, stopping := workload.Scale(500, 20000), 0
		for range n {
			rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 1})
			if err != nil {
				t.Fatal(err)
			}
			release := make(chan struct{})
			op, _ := rt.NewOp("w", impl.Policy{Effect: impl.Pure, Resources: []impl.Req{{Name: "r", Units: 1}}},
				func(ctx context.Context, in int) (int, error) {
					if in == 0 {
						<-release
					}
					return in, nil
				})
			go op.Do(context.Background(), 0)
			for rt.Stats().Started == 0 {
				runtime.Gosched()
			}
			ctx, cancel := context.WithCancel(context.Background())
			errc := make(chan error, 1)
			go func() { _, err := op.Do(ctx, 1); errc <- err }()
			for rt.Stats().Admitted < 2 {
				runtime.Gosched()
			}
			cancel()
			go rt.Shutdown(context.Background())
			if im.Classify(<-errc) == impl.ClassStopping {
				stopping++
			}
			close(release)
			rt.Shutdown(context.Background())
		}
		t.Logf("cancelled waiter reported ErrRuntimeStopping %d/%d times", stopping, n)
		if stopping > 0 && workload.Fixed(im, workload.D6) {
			t.Errorf("cancelled waiter reported ErrRuntimeStopping %d/%d times", stopping, n)
		}
	})
}

// TestD7AttemptContextOnPanic: when a handler panics, its attempt context
// must be cancelled (its timer released); v0.2 leaves it live until the
// deadline.
func TestD7AttemptContextOnPanic(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 1})
		if err != nil {
			t.Fatal(err)
		}
		var attemptCtx context.Context
		op, _ := rt.NewOp("panics", impl.Policy{Effect: impl.Pure, Timeout: time.Hour}, func(ctx context.Context, in int) (int, error) {
			attemptCtx = ctx
			panic("boom")
		})
		func() {
			defer func() { recover() }()
			op.Do(context.Background(), 0)
		}()
		if live := attemptCtx.Err() == nil; live == workload.Fixed(im, workload.D7) {
			t.Errorf("attempt ctx live after panic = %v", live)
		}
		if st := rt.Stats(); st.Finished != st.Admitted {
			t.Errorf("stats after panic: %+v", st)
		}
	})
}

// TestD8ShutdownGoroutines: each Shutdown that times out while draining
// parks a goroutine in v0.2; under the contract repeated calls create none.
func TestD8ShutdownGoroutines(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 1})
		if err != nil {
			t.Fatal(err)
		}
		release, running := make(chan struct{}), make(chan struct{})
		op, _ := rt.NewOp("stuck", impl.Policy{Effect: impl.Pure}, func(ctx context.Context, in int) (int, error) {
			close(running)
			<-release
			return in, nil
		})
		go op.Do(context.Background(), 0)
		<-running
		expired, cancel := context.WithCancel(context.Background())
		cancel()
		rt.Shutdown(expired) // the first call may set up shared drain state
		base := workload.SettledGoroutines()
		const calls = 50
		for range calls {
			if err := rt.Shutdown(expired); err == nil {
				t.Fatal("Shutdown returned nil while a handler was stuck")
			}
		}
		grew := workload.Goroutines(base, 100*time.Millisecond) - base
		t.Logf("%d timed-out Shutdown calls left %+d goroutines", calls, grew)
		if workload.Fixed(im, workload.D8) && grew > 0 {
			t.Errorf("%d timed-out Shutdown calls left %d goroutines", calls, grew)
		}
		close(release)
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Fatal(err)
		}
	})
}

// TestD9StartedOnlyWhenHandlerRuns: a pre-cancelled call with nothing to
// acquire never runs its handler; v0.2 still counts it as Started.
func TestD9StartedOnlyWhenHandlerRuns(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		sc := solo(model.Op{Effect: impl.Pure}, model.Call{Precancel: true, Script: held})
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			r.Start(0)
			res, st := r.Await(0), r.View().Stats
			want := uint64(1)
			if workload.Fixed(im, workload.D9) {
				want = 0
			}
			if res.Class != impl.ClassCanceled || res.Calls != 0 || st.Started != want {
				t.Errorf("got %v, Started %d; want canceled, 0 calls, Started %d", res, st.Started, want)
			}
		})
	})
}

// TestD10PerCallCost: v0.2 starts a watcher goroutine and makes about ten
// allocations per Do; the contract allows no goroutines and the §10
// allocation budgets. Allocation counts are deterministic, so they are
// asserted here; timing is the DoE's business.
func TestD10PerCallCost(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		fixed := workload.Fixed(im, workload.D10)
		rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
		defer rt.Shutdown(context.Background())
		gate := make(chan struct{})
		noop := func(ctx context.Context, in int) (int, error) {
			if in < 0 {
				<-gate
			}
			return in, nil
		}
		one := []impl.Req{{Name: "r", Units: 1}}
		plain, _ := rt.NewOp("plain", impl.Policy{Effect: impl.Pure, Resources: one}, noop)
		timed, _ := rt.NewOp("timed", impl.Policy{Effect: impl.Pure, Resources: one, Timeout: time.Hour}, noop)
		keyed, _ := rt.NewOp("keyed", impl.Policy{Effect: impl.Pure, Resources: one, Key: func(int) string { return "k" }}, noop)
		ctx := context.Background()

		withTimeout := testing.AllocsPerRun(1000, func() { _, c := context.WithTimeout(ctx, time.Hour); c() })
		budgets := []struct {
			name  string
			op    impl.Op
			limit float64
		}{{"plain", plain, 0}, {"timed", timed, withTimeout}, {"keyed", keyed, 1}}
		for _, b := range budgets {
			allocs := testing.AllocsPerRun(5000, func() { b.op.Do(ctx, 1) })
			t.Logf("%-6s Do: %.1f allocs (§10 budget %.0f)", b.name, allocs, b.limit)
			if fixed && allocs > b.limit {
				t.Errorf("%s Do makes %.1f allocs, budget %.0f", b.name, allocs, b.limit)
			}
		}

		const inFlight = 200
		base, started := workload.SettledGoroutines(), rt.Stats().Started
		for range inFlight {
			go plain.Do(ctx, -1)
		}
		for rt.Stats().Started-started < inFlight {
			runtime.Gosched()
		}
		extra := runtime.NumGoroutine() - base - inFlight
		close(gate)
		t.Logf("%d blocked calls: %+d goroutines beyond the callers (%.2f per Do)", inFlight, extra, float64(extra)/inFlight)
		if fixed && extra > 0 {
			t.Errorf("%d goroutines created by %d in-flight calls", extra, inFlight)
		}
	})
}
