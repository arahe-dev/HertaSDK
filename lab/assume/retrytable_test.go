package assume

import (
	"context"
	"fmt"
	"testing"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestRetryTruthTable
//
// Assumption: whether an attempt is retried is a pure function of
// (Effect, Retry.On, MaxAttempts, the attempt's Outcome), fixed at
// construction: unsafe or meaningless contracts never construct, and
// constructed ones retry exactly the safe, requested outcomes (§3.2, §5).
//
// Why it matters: this table is the product's safety claim. A NonIdempotent
// render retried on Uncertain is a double charge; a policy that constructs
// but silently never retries is a reliability bug nobody sees.
//
// Verdict: every cell of Effect (5, incl. unknown and out of range) ×
// On (every subset of the 5 outcomes plus an out-of-range key, plus an
// all-false map) × MaxAttempts (−1, 0, 1, 3) — 1 300 policies — and,
// for each policy that constructs, every handler result (nil, each Fail
// outcome incl. Success and out of range, untagged) matches the model in
// construction error, result class, outcome, handler calls and Retried;
// and all implementations produce the identical table.
func TestRetryTruthTable(t *testing.T) {
	tables := map[string]map[string]string{}
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		tables[im.Name()] = truthTable(t, im)
	})
	var ref string
	for name, tab := range tables {
		if ref == "" {
			ref = name
			continue
		}
		diffs := 0
		for cell, want := range tables[ref] {
			if got := tab[cell]; got != want {
				if diffs++; diffs <= 5 {
					t.Errorf("%s: %s=%s, %s=%s", cell, ref, want, name, got)
				}
			}
		}
		if diffs > 0 {
			t.Errorf("%s and %s disagree on %d of %d cells", ref, name, diffs, len(tables[ref]))
		}
	}
}

var onKeys = []impl.Outcome{impl.Success, impl.Transient, impl.Permanent, impl.Throttled, impl.Uncertain, impl.Outcome(7)}

// handlerResults are the attempt results every cell is tried with.
var handlerResults = []model.Attempt{
	{Return: model.ReturnOK},
	{Return: model.ReturnPlain},
	{Return: model.ReturnTag, Outcome: impl.Success},
	{Return: model.ReturnTag, Outcome: impl.Transient},
	{Return: model.ReturnTag, Outcome: impl.Permanent},
	{Return: model.ReturnTag, Outcome: impl.Throttled},
	{Return: model.ReturnTag, Outcome: impl.Uncertain},
	{Return: model.ReturnTag, Outcome: impl.Outcome(9)},
}

func truthTable(t *testing.T, im impl.Impl) map[string]string {
	specs := []impl.Spec{{Name: "r", Capacity: 1}}
	rt, err := im.NewRuntime(specs...)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Shutdown(context.Background())
	table := map[string]string{}
	constructed := 0
	var calls int
	handler := func(ctx context.Context, in int) (int, error) {
		calls++
		a := handlerResults[in]
		switch a.Return {
		case model.ReturnPlain:
			return 0, workload.ErrScripted
		case model.ReturnTag:
			return 0, im.Fail(a.Outcome, workload.ErrScripted)
		}
		return in, nil
	}
	for _, effect := range []impl.Effect{impl.EffectUnknown, impl.Pure, impl.Idempotent, impl.NonIdempotent, impl.Effect(9)} {
		for mask := 0; mask <= 1<<len(onKeys); mask++ {
			on := onMap(mask)
			for _, attempts := range []int{-1, 0, 1, 3} {
				p := impl.Policy{Effect: effect, Resources: []impl.Req{{Name: "r", Units: 1}}, MaxAttempts: attempts, On: on}
				cell := fmt.Sprintf("effect=%d on=%v attempts=%d", effect, on, attempts)
				constructed++
				op, err := rt.NewOp("cell", p, handler)
				wantClass := model.Validate(specs, "cell", p, false)
				if got := im.Classify(err); got != wantClass {
					t.Fatalf("%s: NewOp error %v (%v), model says %v", cell, got, err, wantClass)
				}
				if err != nil {
					table[cell] = wantClass.String()
					continue
				}
				mop := model.Op{Effect: effect, MaxAttempts: attempts}
				for _, o := range onKeys {
					if on[o] {
						mop.On = append(mop.On, o)
					}
				}
				for i, a := range handlerResults {
					before := rt.Stats().Retried
					calls = 0
					v, err := op.Do(context.Background(), i)
					got := model.Result{Class: im.Classify(err), Outcome: im.OutcomeOf(err), Calls: calls}
					if err == nil {
						got.Value = v
					}
					x := model.NewExec(model.Rules{}, mop, model.Call{Script: model.Script{a}}, i)
					x.Run()
					retried := rt.Stats().Retried - before
					if got != x.Res || retried != uint64(x.Retried) {
						t.Fatalf("%s, handler %v: got %v retried=%d, model %v retried=%d", cell, a, got, retried, x.Res, x.Retried)
					}
					table[fmt.Sprintf("%s handler=%v", cell, a)] = got.String()
				}
			}
		}
	}
	t.Logf("%d policies constructed or refused, %d table entries", constructed, len(table))
	return table
}

// onMap turns a bitmask over onKeys into a Retry.On map. Mask 0 is a nil
// map; the extra mask 1<<len(onKeys) is a map with every key set false.
func onMap(mask int) map[impl.Outcome]bool {
	if mask == 0 {
		return nil
	}
	on := map[impl.Outcome]bool{}
	for i, o := range onKeys {
		on[o] = mask&(1<<i) != 0
	}
	return on
}
