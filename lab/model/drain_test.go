package model_test

import (
	"context"
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestAdmissionDrain races Do against Shutdown on a real scheduler and
// checks the admission/drain state machine (§4 step 1, §6) on the
// recorded trace:
//
//   - the gate is linearizable: once any caller has seen ErrRuntimeStopping
//     from the gate, or Shutdown has returned, every Do that starts later
//     is refused;
//   - Shutdown returns nil only after every admitted handler has exited;
//   - every call is either admitted or refused: Admitted+Rejected = calls;
//   - Finished = Admitted once Shutdown returned nil (D2: immediately under
//     the contract; v0.2 may lag, and the lag is measured).
func TestAdmissionDrain(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rnd := workload.Rand(workload.Seed(t))
		rounds, lagged := workload.Scale(30, 500), 0
		for round := range rounds {
			if drainRound(t, im, workload.Rand(rnd.Uint64())) {
				lagged++
			}
			if t.Failed() {
				t.Fatalf("round %d", round)
			}
		}
		if lagged > 0 && workload.Fixed(im, workload.D2) {
			t.Errorf("Finished < Admitted when Shutdown returned nil in %d/%d rounds (D2)", lagged, rounds)
		}
		t.Logf("Finished lagged Shutdown()==nil in %d/%d rounds", lagged, rounds)
	})
}

func drainRound(t *testing.T, im impl.Impl, rnd *rand.Rand) (lagged bool) {
	rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 2})
	if err != nil {
		t.Fatal(err)
	}
	var entered, exited atomic.Int64
	handler := func(ctx context.Context, in int) (int, error) {
		entered.Add(1)
		defer exited.Add(1)
		time.Sleep(time.Duration(in) * time.Microsecond)
		return in, nil
	}
	gateOnly, _ := rt.NewOp("gate-only", impl.Policy{Effect: impl.Pure}, handler)
	budgeted, _ := rt.NewOp("budgeted", impl.Policy{Effect: impl.Pure, Resources: []impl.Req{{Name: "r", Units: 1}}}, handler)

	var seq atomic.Uint64
	firstRefusal := atomic.Uint64{}
	firstRefusal.Store(math.MaxUint64)
	type rec struct {
		start   uint64
		stopped bool
	}
	var mu sync.Mutex
	var gateRecs []rec
	var calls atomic.Int64

	stopAt := time.Duration(rnd.IntN(2000)) * time.Microsecond
	var shutdownDone atomic.Uint64
	var wg sync.WaitGroup
	for w := range 16 {
		wg.Go(func() {
			for seen := 0; seen < 3; {
				in := w % 20
				calls.Add(1)
				if w%2 == 0 {
					budgeted.Do(context.Background(), in)
					if shutdownDone.Load() != 0 {
						seen++
					}
					continue
				}
				s := seq.Add(1)
				_, err := gateOnly.Do(context.Background(), in)
				stopped := im.Classify(err) == impl.ClassStopping
				if stopped {
					e := seq.Add(1)
					for cur := firstRefusal.Load(); e < cur && !firstRefusal.CompareAndSwap(cur, e); cur = firstRefusal.Load() {
					}
					seen++
				}
				mu.Lock()
				gateRecs = append(gateRecs, rec{s, stopped})
				mu.Unlock()
			}
		})
	}
	time.Sleep(stopAt)
	workload.Within(t, 10*time.Second, "Shutdown", func() {
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
		st := rt.Stats()
		shutdownDone.Store(seq.Add(1))
		if in, out := entered.Load(), exited.Load(); in != out {
			t.Errorf("Shutdown returned nil with %d handlers still running", in-out)
		}
		lagged = st.Finished < st.Admitted
	})
	workload.Within(t, 10*time.Second, "callers after Shutdown", wg.Wait)

	cutoff := min(firstRefusal.Load(), shutdownDone.Load())
	for _, r := range gateRecs {
		if r.start > cutoff && !r.stopped {
			t.Errorf("call started at seq %d was admitted after stopping was observed at seq %d", r.start, cutoff)
			break
		}
	}
	if st := rt.Stats(); st.Admitted+st.Rejected != uint64(calls.Load()) || st.Finished != st.Admitted {
		t.Errorf("stats %+v for %d calls", st, calls.Load())
	}
	return lagged
}

// TestShutdownIdleExpired: on an idle runtime, Shutdown with an
// already-expired ctx has nothing to wait for and must return nil (§6,
// "drained beats expired"). v0.2 picks at random (DNondet); the rate is
// measured.
func TestShutdownIdleExpired(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		n, nils := workload.Scale(2000, 50000), 0
		for range n {
			rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 1})
			if err != nil {
				t.Fatal(err)
			}
			if rt.Shutdown(ctx) == nil {
				nils++
			}
		}
		t.Logf("idle Shutdown(expired ctx) returned nil %d/%d times", nils, n)
		if workload.Fixed(im, workload.DNondet) && nils != n {
			t.Errorf("returned ctx.Err() %d/%d times on an idle runtime", n-nils, n)
		}
	})
}

// TestShutdownConcurrentCalls: many concurrent Shutdown calls all block
// while a handler runs, then all return nil (§6: idempotent, safe to call
// concurrently). Timed-out calls return ctx.Err() and draining continues.
func TestShutdownConcurrentCalls(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 1})
		if err != nil {
			t.Fatal(err)
		}
		release, running := make(chan struct{}), make(chan struct{})
		op, _ := rt.NewOp("slow", impl.Policy{Effect: impl.Pure, Resources: []impl.Req{{Name: "r", Units: 1}}},
			func(ctx context.Context, in int) (int, error) {
				close(running)
				<-release
				return in, nil
			})
		go op.Do(context.Background(), 0)
		<-running

		const callers = 16
		results := make(chan error, callers)
		for range callers {
			go func() { results <- rt.Shutdown(context.Background()) }()
		}
		short, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()
		if err := rt.Shutdown(short); im.Classify(err) != impl.ClassDeadline {
			t.Errorf("Shutdown with a 5ms ctx while draining: %v, want deadline exceeded", err)
		}
		select {
		case err := <-results:
			t.Fatalf("a Shutdown returned (%v) while a handler was still running", err)
		default:
		}
		close(release)
		workload.Within(t, 5*time.Second, "concurrent Shutdown", func() {
			for range callers {
				if err := <-results; err != nil {
					t.Errorf("Shutdown: %v", err)
				}
			}
		})
	})
}

// TestFinishedVisibleAtShutdown hammers the D2 window: Shutdown waits on
// one call that finishes just as the drain completes. Under the contract
// Stats().Finished == Admitted must already hold when Shutdown returns
// nil; v0.2 counts Finished after its drain signal, and the rate at which
// that shows is measured.
func TestFinishedVisibleAtShutdown(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		n, lagged := workload.Scale(20000, 1000000), 0
		for range n {
			rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 1})
			if err != nil {
				t.Fatal(err)
			}
			running := make(chan struct{})
			op, _ := rt.NewOp("brief", impl.Policy{Effect: impl.Pure}, func(ctx context.Context, in int) (int, error) {
				close(running)
				return in, nil
			})
			go op.Do(context.Background(), 0)
			<-running
			if err := rt.Shutdown(context.Background()); err != nil {
				t.Fatal(err)
			}
			if st := rt.Stats(); st.Finished != st.Admitted {
				lagged++
			}
		}
		t.Logf("Finished < Admitted right after Shutdown()==nil: %d/%d", lagged, n)
		if lagged > 0 && workload.Fixed(im, workload.D2) {
			t.Errorf("Finished lagged Shutdown()==nil %d/%d times (D2)", lagged, n)
		}
	})
}
