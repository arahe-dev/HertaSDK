package model

import (
	"fmt"
	"slices"
	"strings"
)

// Minimize greedily drops steps from sc while it stays valid under both
// rule sets and fails still reports true. The result is 1-minimal: removing
// any single remaining step (with the steps that depend on it) loses the
// failure.
func Minimize(sc Scenario, fails func(Scenario) bool) Scenario {
	for changed := true; changed; {
		changed = false
		for i := len(sc.Steps) - 1; i >= 0; i-- {
			cand := sc
			cand.Steps = withoutStep(sc.Steps, i)
			if Check(Rules{}, &cand) == nil && Check(Rules{Fixed: true}, &cand) == nil && fails(cand) {
				sc, changed = cand, true
				i = min(i, len(sc.Steps))
			}
		}
	}
	return sc
}

// withoutStep drops step i; dropping a Start also drops the call's later steps.
func withoutStep(steps []Step, i int) []Step {
	out := slices.Delete(slices.Clone(steps), i, i+1)
	if steps[i].Kind == StepStart {
		out = slices.DeleteFunc(out, func(s Step) bool { return s.Kind != StepShutdown && s.Call == steps[i].Call })
	}
	return out
}

// Format renders sc as a readable reproduction.
func Format(sc Scenario) string {
	var b strings.Builder
	fmt.Fprintf(&b, "resources: %v\n", sc.Specs)
	for _, op := range sc.Ops {
		fmt.Fprintf(&b, "%v\n", op)
	}
	started := map[int]bool{}
	for _, st := range sc.Steps {
		if st.Kind == StepStart {
			started[st.Call] = true
		}
	}
	for i, c := range sc.Calls {
		if started[i] {
			fmt.Fprintf(&b, "c%d: op%d key=%q precancel=%v deadline=%v script=%v\n", i, c.Op, c.Key, c.Precancel, c.Deadline, c.Script)
		}
	}
	for i, st := range sc.Steps {
		fmt.Fprintf(&b, "%2d. %v\n", i, st)
	}
	return b.String()
}
