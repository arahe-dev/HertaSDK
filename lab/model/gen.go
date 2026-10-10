package model

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// GenConfig shapes random scenarios.
type GenConfig struct {
	Steps     int      // target length
	Resources int      // at most 3
	MaxCap    int64    // capacities are drawn from 1..MaxCap
	Ops       int      // operations declared
	Keys      []string // key pool for keyed ops ("" = unserialized call)
	Shutdowns int      // at most this many Shutdown steps
}

// Generate draws a scenario whose every step is applicable and
// unambiguous under both v0.2 and fixed rules, so any implementation can
// be compared with its model at every step.
func Generate(rnd *rand.Rand, g GenConfig) Scenario {
	sc := Scenario{Specs: genSpecs(rnd, g)}
	for i := range g.Ops {
		sc.Ops = append(sc.Ops, genOp(rnd, i, sc.Specs))
	}
	for len(sc.Steps) < g.Steps {
		phases := mustReplay(&sc).View().Phases
		placed := false
		for range 8 {
			next := Scenario{Specs: sc.Specs, Ops: sc.Ops, Calls: slices.Clip(sc.Calls), Steps: slices.Clip(sc.Steps)}
			next.Steps = append(next.Steps, g.step(rnd, &next, phases))
			if Check(Rules{}, &next) == nil && Check(Rules{Fixed: true}, &next) == nil {
				sc, placed = next, true
				break
			}
		}
		if !placed {
			break
		}
	}
	return sc
}

func mustReplay(sc *Scenario) *System {
	s := NewSystem(Rules{}, sc)
	for _, st := range sc.Steps {
		if err := s.Apply(st); err != nil {
			panic(err)
		}
	}
	return s
}

func (g GenConfig) step(rnd *rand.Rand, sc *Scenario, phases []Phase) Step {
	shutdowns := 0
	for _, st := range sc.Steps {
		if st.Kind == StepShutdown {
			shutdowns++
		}
	}
	pick := func(want ...Phase) (int, bool) {
		var ids []int
		for i, p := range phases {
			if slices.Contains(want, p) {
				ids = append(ids, i)
			}
		}
		if len(ids) == 0 {
			return 0, false
		}
		return ids[rnd.IntN(len(ids))], true
	}
	switch n := rnd.IntN(100); {
	case n < 30:
		if id, ok := pick(Holding); ok {
			return Step{Kind: StepRelease, Call: id}
		}
	case n < 42:
		if id, ok := pick(Parked, Holding); ok {
			return Step{Kind: StepCancel, Call: id}
		}
	case n < 46 && shutdowns < g.Shutdowns && len(sc.Steps) >= g.Steps*2/3:
		return Step{Kind: StepShutdown, Expired: rnd.IntN(3) == 0}
	}
	sc.Calls = append(sc.Calls, genCall(rnd, sc.Ops, g.Keys))
	return Step{Kind: StepStart, Call: len(sc.Calls) - 1}
}

func genSpecs(rnd *rand.Rand, g GenConfig) []impl.Spec {
	n := 1 + rnd.IntN(min(g.Resources, 3))
	specs := make([]impl.Spec, n)
	for i := range specs {
		specs[i] = impl.Spec{Name: string(rune('a' + i)), Capacity: 1 + rnd.Int64N(g.MaxCap)}
	}
	return specs
}

func genOp(rnd *rand.Rand, i int, specs []impl.Spec) Op {
	op := Op{
		Name:        fmt.Sprintf("op%d", i),
		Effect:      impl.Effects[rnd.IntN(len(impl.Effects))],
		Keyed:       rnd.IntN(5) < 2,
		MaxAttempts: rnd.IntN(4),
	}
	if rnd.IntN(10) < 3 {
		op.Admission = impl.Reject
	}
	if rnd.IntN(2) == 0 {
		op.Timeout = time.Hour // exercises the WithTimeout path; never fires
	}
	// Shuffled so that claim order differs from the global name order.
	for _, j := range rnd.Perm(len(specs)) {
		if rnd.IntN(10) < 6 {
			op.Resources = append(op.Resources, impl.Req{Name: specs[j].Name, Units: 1 + rnd.Int64N(specs[j].Capacity)})
		}
	}
	if rnd.IntN(2) == 0 {
		op.On = append(op.On, impl.Transient)
	}
	if op.Effect != impl.NonIdempotent && rnd.IntN(2) == 0 {
		op.On = append(op.On, impl.Uncertain)
	}
	return op
}

func genCall(rnd *rand.Rand, ops []Op, keys []string) Call {
	c := Call{Op: rnd.IntN(len(ops))}
	op := ops[c.Op]
	if op.Keyed && len(keys) > 0 {
		c.Key = keys[rnd.IntN(len(keys))]
	}
	c.Precancel = rnd.IntN(20) == 0
	first := genAttempt(rnd)
	switch n := rnd.IntN(20); {
	case n < 11:
		first.Block = BlockHold
	case n < 14:
		first.Block = BlockHoldCtx
	}
	c.Script = Script{first}
	for range rnd.IntN(3) {
		c.Script = append(c.Script, genAttempt(rnd))
	}
	return c
}

func genAttempt(rnd *rand.Rand) Attempt {
	var a Attempt
	switch n := rnd.IntN(100); {
	case n < 45:
		a.Return = ReturnOK
	case n < 75:
		a.Return = ReturnTag
		a.Outcome = impl.Outcome(rnd.IntN(len(impl.Outcomes) + 1)) // +1: out of range
	case n < 85:
		a.Return = ReturnPlain
	case n < 95:
		a.Return = ReturnCtxErr
	default:
		a.Return = ReturnPanic
	}
	a.Cancel = rnd.IntN(25) == 0
	return a
}
