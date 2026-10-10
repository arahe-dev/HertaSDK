package system_test

import (
	"testing"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestDifferential replays seeded random scenarios (resources, weights,
// keys, Wait/Reject, retries, scripted outcomes, panics, cancellations,
// Shutdown) on every implementation and checks every observable — each
// call's phase and result, Stats, each Shutdown — against the model of the
// rules that implementation claims, after every step.
//
// Implementations are compared through the model: two implementations may
// differ only where the v0.2 and fixed models differ, i.e. only by a §8
// delta. Any other difference fails with a minimized reproduction.
func TestDifferential(t *testing.T) {
	seed := workload.Seed(t)
	rnd := workload.Rand(seed)
	// Wide scenarios exercise multi-resource claims and keys; tight ones (unit
	// capacities, one key) maximize queueing, hand-off and cascades.
	cfgs := []model.GenConfig{
		{Steps: 40, Resources: 3, MaxCap: 4, Ops: 4, Keys: []string{"", "k1", "k2"}, Shutdowns: 2},
		{Steps: 60, Resources: 2, MaxCap: 1, Ops: 3, Keys: []string{"k"}, Shutdowns: 1},
	}
	n := workload.Scale(3000, 100000)

	var steps, deltaScenarios int
	classes := map[impl.Class]int{}
	for i := range n {
		sc := model.Generate(rnd, cfgs[i%len(cfgs)])
		steps += len(sc.Steps)
		v2, fixed := finalView(model.Rules{}, &sc), finalView(model.Rules{Fixed: true}, &sc)
		if model.Diff(model.Rules{}, v2, fixed) != "" {
			deltaScenarios++
		}
		for i, p := range v2.Phases {
			if p == model.Done {
				classes[v2.Results[i].Class]++
			}
		}
		for _, im := range workload.Impls(t) {
			if d := workload.Replay(t, im, sc); d != nil {
				small := model.Minimize(sc, func(c model.Scenario) bool { return workload.Replay(t, im, c) != nil })
				t.Fatalf("%s diverged from its model at %v\nminimized reproduction:\n%s\nfirst divergence there: %v",
					im.Name(), d, model.Format(small), workload.Replay(t, im, small))
			}
		}
	}
	t.Logf("%d scenarios, %d steps; %d scenarios (%.1f%%) end differently under the §8 deltas",
		n, steps, deltaScenarios, 100*float64(deltaScenarios)/float64(n))
	t.Logf("results by class (v0.2 model): %v", classes)
}

func finalView(r model.Rules, sc *model.Scenario) model.View {
	s := model.NewSystem(r, sc)
	for _, st := range sc.Steps {
		if err := s.Apply(st); err != nil {
			panic(err) // Generate only emits scenarios valid under both rules
		}
	}
	return s.View()
}

// TestDifferentialPower is the lab's self-test: judged by the fixed rules,
// v0.2 must be caught in every scenario where the two models disagree. A
// harness that cannot see the known §8 defects cannot vouch for recon.
func TestDifferentialPower(t *testing.T) {
	if !selected(t, impl.Orig) {
		t.Skip("needs orig")
	}
	seed := workload.Seed(t)
	rnd := workload.Rand(seed)
	cfg := model.GenConfig{Steps: 30, Resources: 2, MaxCap: 3, Ops: 3, Keys: []string{"", "k1"}, Shutdowns: 2}
	var caught int
	var smallest *model.Scenario
	for range workload.Scale(200, 2000) {
		sc := model.Generate(rnd, cfg)
		if !modelsDisagree(sc) {
			continue
		}
		if workload.ReplayAs(t, impl.Orig, model.Rules{Fixed: true}, sc) == nil {
			t.Fatalf("v0.2 passed as fixed in a scenario where the models disagree:\n%s", model.Format(sc))
		}
		caught++
		if smallest == nil || len(sc.Steps) < len(smallest.Steps) {
			small := model.Minimize(sc, modelsDisagree)
			smallest = &small
		}
	}
	if caught == 0 {
		t.Fatal("no scenario exercised a §8 delta; the generator has lost its reach")
	}
	t.Logf("v0.2 caught in all %d delta-exercising scenarios; smallest:\n%s", caught, model.Format(*smallest))
}

// modelsDisagree reports whether the v0.2 and fixed models differ at any step.
func modelsDisagree(sc model.Scenario) bool {
	v2, fixed := model.NewSystem(model.Rules{}, &sc), model.NewSystem(model.Rules{Fixed: true}, &sc)
	for _, st := range sc.Steps {
		if v2.Apply(st) != nil || fixed.Apply(st) != nil {
			return false
		}
		if model.Diff(model.Rules{Fixed: true}, fixed.View(), v2.View()) != "" {
			return true
		}
	}
	return false
}

func selected(t *testing.T, want impl.Impl) bool {
	for _, im := range workload.Impls(t) {
		if im.Name() == want.Name() {
			return true
		}
	}
	return false
}
