package assume

import (
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// solo builds a resource-free, unkeyed op scenario with one call.
func solo(op model.Op, c model.Call) model.Scenario {
	op.Name = "solo"
	return model.Scenario{Specs: []impl.Spec{{Name: "unused", Capacity: 1}}, Ops: []model.Op{op}, Calls: []model.Call{c}}
}

// TestFreshDeadlinePerAttempt
//
// Assumption: every attempt gets its own Timeout, derived from the
// caller's context; retries do not share one deadline (§5 step 2).
//
// Why it matters: with a shared deadline, attempt 2 starts already expired
// and a configured retry silently never happens (v0.2 review mutant M16
// survived the shipped suite).
//
// Verdict: a handler that waits out its deadline three times takes exactly
// 3×30ms of virtual time, makes 3 calls and reports Uncertain.
func TestFreshDeadlinePerAttempt(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		op := model.Op{Effect: impl.Pure, Timeout: 30 * time.Millisecond, MaxAttempts: 3, On: []impl.Outcome{impl.Uncertain}}
		sc := solo(op, model.Call{Script: model.Script{{Block: model.BlockDeadline, Return: model.ReturnCtxErr}}})
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			r.Start(0)
			res := r.Await(0)
			if res.Calls != 3 || res.Outcome != impl.Uncertain || r.Elapsed(0) != 90*time.Millisecond {
				t.Errorf("got %v in %v; want 3 calls, Uncertain, in exactly 90ms", res, r.Elapsed(0))
			}
		})
	})
}

// TestCooperativeTimeout
//
// Assumption: Timeout is cooperative: a handler that ignores its context
// runs to completion, and its late result is returned, not discarded (§5
// step 3, v0.2 behavior kept).
//
// Why it matters: callers must not read Timeout as a hard bound on Do
// (worst case is MaxAttempts × the handler's own duration), and a late
// success must not be reported as a failure that invites a retry.
//
// Verdict: with Timeout 10ms, a handler that sleeps 50ms and succeeds
// returns its value after exactly 50ms, with one call.
func TestCooperativeTimeout(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		op := model.Op{Effect: impl.Idempotent, Timeout: 10 * time.Millisecond, MaxAttempts: 3,
			On: []impl.Outcome{impl.Transient, impl.Uncertain}}
		sc := solo(op, model.Call{Script: model.Script{{Block: model.BlockSleep, Sleep: 50 * time.Millisecond}}})
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			r.Start(0)
			want := model.Result{Class: impl.ClassNil, Outcome: impl.Success, Value: 0, Calls: 1}
			if res := r.Await(0); res != want || r.Elapsed(0) != 50*time.Millisecond {
				t.Errorf("got %v in %v; want %v in exactly 50ms", res, r.Elapsed(0), want)
			}
		})
	})
}
