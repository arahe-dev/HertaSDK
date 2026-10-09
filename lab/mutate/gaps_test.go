package mutate_test

// Gap tests: each one kills a mutant that survived every other suite (see
// lab/mutate/results). They are implementation-agnostic, so they run
// against every implementation in LAB_IMPLS like the rest of the lab.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestSynctestProbe runs only as the child of workload.RequireBubbles.
func TestSynctestProbe(t *testing.T) { workload.Probe(t) }

func newOp(t *testing.T, rt impl.Runtime, p impl.Policy, h impl.Handler) impl.Op {
	t.Helper()
	if p.Effect == impl.EffectUnknown {
		p.Effect = impl.Idempotent
	}
	op, err := rt.NewOp(t.Name(), p, h)
	if err != nil {
		t.Fatal(err)
	}
	return op
}

func newRuntime(t *testing.T, im impl.Impl, specs ...impl.Spec) impl.Runtime {
	t.Helper()
	rt, err := im.NewRuntime(specs...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return rt
}

var one = []impl.Req{{Name: "r", Units: 1}}

// TestGapTimeoutBoundary: §3.4 says Timeout ≤ 0 means no per-attempt
// deadline, and any positive Timeout means one. Kills recon's
// max(Timeout, 0) → max(Timeout, -1) (a negative Timeout became an
// already-expired deadline) and v0.2's Timeout > 0 → Timeout > 1 (a 1 ns
// Timeout set no deadline).
func TestGapTimeoutBoundary(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rt := newRuntime(t, im, impl.Spec{Name: "r", Capacity: 1})
		for _, c := range []struct {
			timeout  time.Duration
			deadline bool
		}{{-time.Second, false}, {-1, false}, {0, false}, {1, true}, {time.Hour, true}} {
			op := newOp(t, rt, impl.Policy{Resources: one, Timeout: c.timeout},
				func(ctx context.Context, _ int) (int, error) {
					_, ok := ctx.Deadline()
					return map[bool]int{false: 0, true: 1}[ok], nil
				})
			got, err := op.Do(context.Background(), 0)
			if err != nil || (got == 1) != c.deadline {
				t.Errorf("Timeout %v: handler saw deadline=%v (err %v), want %v", c.timeout, got == 1, err, c.deadline)
			}
		}
	})
}

// TestGapFailureKeepsErrorText: a tagged error reads exactly like the
// error it wraps, through Fail and through Do. Kills v0.2's
// Failure.Error `f.Err == nil` → `!=` (wrong text, then a nil dereference).
func TestGapFailureKeepsErrorText(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		boom := errors.New("boom")
		for _, o := range []impl.Outcome{impl.Transient, impl.Permanent, impl.Throttled, impl.Uncertain} {
			if got := im.Fail(o, boom).Error(); got != "boom" {
				t.Errorf("Fail(%d, boom).Error() = %q, want %q", o, got, "boom")
			}
		}
		rt := newRuntime(t, im, impl.Spec{Name: "r", Capacity: 1})
		op := newOp(t, rt, impl.Policy{Resources: one}, func(context.Context, int) (int, error) {
			return 0, im.Fail(impl.Permanent, boom)
		})
		if _, err := op.Do(context.Background(), 0); err == nil || err.Error() != "boom" || !errors.Is(err, boom) {
			t.Errorf("Do error = %v, want the handler's tagged boom", err)
		}
	})
}

// TestGapAcquisitionFollowsNameOrder: §4 step 3 acquires in ascending
// name order whatever order the runtime's specs or the policy's
// requirements are declared in, so a call parked on "a" holds nothing of
// "b" or "c". Kills v0.2's deleted sortStrings(order) and the eight
// mutants of its insertion sort that leave names unsorted: acquisition
// then followed declaration order, still deadlock-free but holding
// capacity while parked.
func TestGapAcquisitionFollowsNameOrder(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		workload.RequireBubbles(t, im)
		synctest.Test(t, func(t *testing.T) {
			// Declared b, a, c: every partial insertion sort leaves b before a.
			rt := newRuntime(t, im, impl.Spec{Name: "b", Capacity: 1}, impl.Spec{Name: "a", Capacity: 1},
				impl.Spec{Name: "c", Capacity: 1})
			gate := make(chan struct{})
			held := func(context.Context, int) (int, error) { <-gate; return 0, nil }
			free := func(context.Context, int) (int, error) { return 0, nil }
			claim := func(names ...string) []impl.Req {
				var rs []impl.Req
				for _, n := range names {
					rs = append(rs, impl.Req{Name: n, Units: 1})
				}
				return rs
			}
			holdA := newOp(t, rt, impl.Policy{Resources: claim("a")}, held)
			all := newOp(t, rt, impl.Policy{Resources: claim("c", "b", "a")}, free)

			go holdA.Do(context.Background(), 0)
			synctest.Wait()
			go all.Do(context.Background(), 0) // parks on "a"
			synctest.Wait()
			for _, n := range []string{"b", "c"} {
				probe := newOp(t, rt, impl.Policy{Resources: claim(n), Admission: impl.Reject}, free)
				if _, err := probe.Do(context.Background(), 0); err != nil {
					t.Errorf("%s is held by a call parked on a (%v): acquisition did not follow name order", n, err)
				}
			}
			close(gate)
			synctest.Wait()
		})
	})
}

// TestGapNothingAccumulates: per-call state is released on every path, so
// a long-lived process holds nothing per past call. Each case warms up,
// then runs 50 000 calls and allows 16 B of live heap per call: enough for
// v0.2's one-off goroutine bookkeeping (about 100 KB, not growing), far
// below the 100 B or more per call that a retained context or key entry
// costs. Kills v0.2's deleted `defer cancelAdmission()` (every Do left a
// child context attached to a long-lived caller context) and its deleted
// `kl.release(l)` on the failed-acquire path (every cancelled key waiter
// leaked its key entry).
func TestGapNothingAccumulates(t *testing.T) {
	const warm, calls, perCall = 2_000, 50_000, 16
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rt := newRuntime(t, im, impl.Spec{Name: "r", Capacity: 1})
		free := func(context.Context, int) (int, error) { return 0, nil }
		parent, cancel := context.WithCancel(context.Background()) // a server's base context
		defer cancel()
		dead, kill := context.WithCancel(parent)
		kill()
		cases := []struct {
			name string
			ctx  context.Context
			op   impl.Op
		}{
			{"live parent", parent, newOp(t, rt, impl.Policy{Resources: one}, free)},
			{"live parent, timed", parent, newOp(t, rt, impl.Policy{Resources: one, Timeout: time.Hour}, free)},
			{"cancelled caller, distinct keys", dead, newOp(t, rt,
				impl.Policy{Resources: one, Key: func(i int) string { return fmt.Sprint("key-", i) }}, free)},
		}
		for _, c := range cases {
			for i := range warm {
				c.op.Do(c.ctx, -i)
			}
			before := workload.LiveHeap()
			for i := range calls {
				c.op.Do(c.ctx, i)
			}
			after := workload.LiveHeap()
			if after > before && (after-before)/calls > perCall {
				t.Errorf("%s: live heap grew %d B over %d calls (%d B/call), want ≤ %d B/call",
					c.name, after-before, calls, (after-before)/calls, perCall)
			}
		}
	})
}

// TestGapCancelStorm is meant for -race (the cascade's race-lab stage):
// key and unit waiters give up at random while others hand off, and
// Shutdown closes every queue mid-storm. Its own assertion is only the
// drain contract (§6, Δ2); its purpose is to put every waiter path on a
// second goroutine so the race detector sees a lock that a mutant drops.
// Kills five mutants that only drop a lock: recon's inlined
// `defer kl.mu.Unlock()` in KeyLocker.leave and its `closed = true` hoisted
// above the lock in KeyLocker.close and Resource.close; v0.2's `l.refs++`
// moved after kl.mu.Unlock() and its inlined `defer rt.mu.RUnlock()` in
// begin (wg.Add racing Shutdown's Wait).
func TestGapCancelStorm(t *testing.T) {
	const workers, calls = 16, 150
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 2})
		if err != nil {
			t.Fatal(err)
		}
		work := func(context.Context, int) (int, error) { time.Sleep(20 * time.Microsecond); return 0, nil }
		ops := []impl.Op{ // even workers queue for keys, odd ones for units
			newOp(t, rt, impl.Policy{Resources: one, Key: func(i int) string { return fmt.Sprint("k", i%3) }}, work),
			newOp(t, rt, impl.Policy{Resources: one}, work),
		}
		rnd := workload.Rand(workload.Seed(t))
		patience := make([]time.Duration, workers*calls)
		for i := range patience {
			patience[i] = time.Duration(rnd.IntN(200)) * time.Microsecond
		}
		var wg sync.WaitGroup
		for w := range workers {
			wg.Go(func() {
				for c := range calls {
					i := w*calls + c
					ctx, cancel := context.WithTimeout(context.Background(), patience[i])
					ops[w%2].Do(ctx, i)
					cancel()
				}
			})
		}
		time.Sleep(5 * time.Millisecond)
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Fatal(err)
		}
		if s := rt.Stats(); s.Finished != s.Admitted && workload.Fixed(im, workload.D2) {
			t.Errorf("Shutdown returned nil with Finished %d != Admitted %d", s.Finished, s.Admitted)
		}
		wg.Wait()
	})
}
