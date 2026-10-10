package model_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"testing/synctest"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestRetryEngine isolates the retry loop (§5): resource-free, unkeyed
// operations run scripted attempt sequences — successes, tagged and plain
// failures, panics, handlers that wait for their deadline or sleep past
// it, handlers that cancel their own caller — under per-attempt timeouts
// and caller deadlines. In virtual time every call is exact, so each one
// must match the model in error class, outcome, value, handler calls,
// Retried, Started and elapsed time to the nanosecond.
func TestRetryEngine(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		workload.RequireBubbles(t, im)
		rnd := workload.Rand(workload.Seed(t))
		rules := workload.Rules(im)
		var calls, deltaCalls int
		for range workload.Scale(40, 1000) {
			sc := retryScenario(rnd, 8, 50)
			calls += len(sc.Calls)
			deltaCalls += countDisagreements(sc)
			synctest.Test(t, func(t *testing.T) { driveRetry(t, im, rules, &sc) })
			if t.Failed() {
				return
			}
		}
		t.Logf("%d calls, %d of them (%.1f%%) where v0.2 and the contract disagree (D3, D4)",
			calls, deltaCalls, 100*float64(deltaCalls)/float64(calls))
	})
}

func driveRetry(t *testing.T, im impl.Impl, rules model.Rules, sc *model.Scenario) {
	r, err := workload.NewRig(im, sc)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	}()
	sys := model.NewSystem(rules, sc)
	for i := range sc.Calls {
		if err := sys.Apply(model.Step{Kind: model.StepStart, Call: i}); err != nil {
			t.Fatal(err)
		}
		r.Start(i)
		got := r.Await(i)
		want := sys.View()
		if got != want.Results[i] || r.Elapsed(i) != time.Duration(sys.Elapsed(i)) || r.Runtime().Stats() != want.Stats {
			t.Fatalf("c%d (%v, deadline %v, precancel %v, script %v):\n got  %v in %v, stats %+v\n want %v in %v, stats %+v",
				i, sc.Ops[sc.Calls[i].Op], sc.Calls[i].Deadline, sc.Calls[i].Precancel, sc.Calls[i].Script,
				got, r.Elapsed(i), r.Runtime().Stats(), want.Results[i], time.Duration(sys.Elapsed(i)), want.Stats)
		}
	}
}

// retryScenario draws ops and solo calls. Durations are chosen so no two
// timers can fire at the same instant: attempt timeouts and sleeps are
// distinct whole milliseconds, caller deadlines end in .5ms.
func retryScenario(rnd *rand.Rand, ops, calls int) model.Scenario {
	sc := model.Scenario{Specs: []impl.Spec{{Name: "unused", Capacity: 1}}}
	for i := range ops {
		op := model.Op{Name: fmt.Sprintf("op%d", i), Effect: impl.Effects[rnd.IntN(3)], MaxAttempts: rnd.IntN(5)}
		if rnd.IntN(3) > 0 {
			op.Timeout = 10 * time.Millisecond
		}
		if rnd.IntN(3) > 0 {
			op.On = append(op.On, impl.Transient)
		}
		if op.Effect != impl.NonIdempotent && rnd.IntN(3) > 0 {
			op.On = append(op.On, impl.Uncertain)
		}
		sc.Ops = append(sc.Ops, op)
	}
	deadlines := []time.Duration{0, 0, 0, 25500 * time.Microsecond, 41500 * time.Microsecond}
	for range calls {
		c := model.Call{Op: rnd.IntN(ops), Deadline: deadlines[rnd.IntN(len(deadlines))], Precancel: rnd.IntN(25) == 0}
		timed := sc.Ops[c.Op].Timeout > 0 || c.Deadline > 0
		for range 1 + rnd.IntN(4) {
			c.Script = append(c.Script, retryAttempt(rnd, timed))
		}
		sc.Calls = append(sc.Calls, c)
	}
	for i := range sc.Calls {
		sc.Steps = append(sc.Steps, model.Step{Kind: model.StepStart, Call: i})
	}
	return sc
}

func retryAttempt(rnd *rand.Rand, timed bool) model.Attempt {
	var a model.Attempt
	switch n := rnd.IntN(10); {
	case n < 3 && timed:
		a.Block = model.BlockDeadline
	case n < 5:
		a.Block = model.BlockSleep
		a.Sleep = []time.Duration{3 * time.Millisecond, 17 * time.Millisecond}[rnd.IntN(2)]
	}
	a.Cancel = rnd.IntN(15) == 0
	switch n := rnd.IntN(20); {
	case n < 5:
		a.Return = model.ReturnOK
	case n < 11:
		a.Return = model.ReturnTag
		a.Outcome = impl.Outcome(rnd.IntN(6))
	case n < 15:
		a.Return = model.ReturnPlain
	case n < 19:
		a.Return = model.ReturnCtxErr
	default:
		a.Return = model.ReturnPanic
	}
	return a
}

// countDisagreements counts calls whose result differs between the models.
func countDisagreements(sc model.Scenario) int {
	v2, fixed := model.NewSystem(model.Rules{}, &sc), model.NewSystem(model.Rules{Fixed: true}, &sc)
	n := 0
	for i, st := range sc.Steps {
		v2.Apply(st)
		fixed.Apply(st)
		if v2.View().Results[i] != fixed.View().Results[i] {
			n++
		}
	}
	return n
}
