package doe

import (
	"math"
	"math/rand/v2"
	"testing"
)

// synthetic builds trials of a crossed Screen32 experiment whose raw
// response is y = 10 + 3A - 2B + 1.5AB + 4X + 1·XC + N(0, σ²), with effects
// stated as half-differences (DoE "coefficients"), so each true effect is
// twice its coefficient.
func synthetic(reps int, sigma float64, seed uint64) (*Experiment, []Trial) {
	arms := []Arm{{Label: "base"}, {Label: "cand"}}
	x := &Experiment{Kind: "screen", Factors: Screening, Design: Screen32, Arms: arms, Reps: reps}
	x.Points = make([]Point, Screen32.Runs())
	rng := rand.New(rand.NewPCG(seed, 7))
	var trials []Trial
	for rep := range reps {
		for p := range x.Points {
			for a := range arms {
				l := func(f int) float64 { return float64(Screen32.Level(p, f)) }
				xi := float64(2*a - 1)
				y := 10 + 3*l(0) - 2*l(1) + 1.5*l(0)*l(1) + 4*xi + 1*xi*l(2) + sigma*rng.NormFloat64()
				trials = append(trials, Trial{Rep: rep, Point: p, Arm: a, Res: Result{AllocsPerOp: y}})
			}
		}
	}
	return x, trials
}

func pairEffects(an Analysis, resp string) map[string]Effect {
	m := map[string]Effect{}
	for _, e := range an.Effects {
		if e.Response == resp && e.Scope == "base vs cand" {
			m[e.Aliases[0]] = e
		}
	}
	return m
}

func TestAnalyzeRecoversKnownEffects(t *testing.T) {
	x, trials := synthetic(3, 0.5, 1)
	eff := pairEffects(Analyze(x, trials), "allocs_op")
	want := map[string]float64{"A": 6, "B": -4, "AB": 3, "X": 8, "CX": 2}
	for name, w := range want {
		e, ok := eff[name]
		if !ok {
			t.Fatalf("no effect %q", name)
		}
		if math.Abs(e.Estimate-w) > 4*e.SE {
			t.Errorf("%s = %.3f ± %.3f, want %v", name, e.Estimate, e.SE, w)
		}
		if !e.Sig || e.Lenth == "" {
			t.Errorf("%s not flagged: p=%.2g lenth=%q", name, e.P, e.Lenth)
		}
	}
	// SE = 2σ/√N with N = 32·2·3 trials.
	if se := eff["A"].SE; math.Abs(se-2*0.5/math.Sqrt(192)) > 0.02 {
		t.Errorf("SE %.4f, want ≈ %.4f", se, 2*0.5/math.Sqrt(192))
	}
	var falsePos int
	for name, e := range eff {
		if _, real := want[name]; !real && e.Sig {
			falsePos++
			t.Logf("false positive %s = %.3f (p=%.2g)", e.Name(), e.Estimate, e.P)
		}
	}
	if falsePos > 1 {
		t.Errorf("%d false positives among null effects", falsePos)
	}
}

func TestComparisonReadsTheImplFactor(t *testing.T) {
	x, trials := synthetic(3, 0.5, 2)
	an := Analyze(x, trials)
	for _, c := range an.Comparisons {
		if c.Response != "allocs_op" {
			continue
		}
		// Candidate is 8 allocs higher; fewer allocs are better.
		if c.Verdict != "worse" || c.Lo > 8 || c.Hi < 8 {
			t.Errorf("verdict %s CI [%.2f, %.2f]", c.Verdict, c.Lo, c.Hi)
		}
		if len(c.Interactions) == 0 || c.Interactions[0].Aliases[0] != "CX" {
			t.Errorf("interactions %v", c.Interactions)
		}
		if math.Abs(c.CandMean-c.BaseMean-8) > 0.5 {
			t.Errorf("means %v %v", c.BaseMean, c.CandMean)
		}
		return
	}
	t.Fatal("no allocs_op comparison")
}

func TestPureNoiseRarelySignificant(t *testing.T) {
	// A/A in miniature: no effects at all. Bonferroni keeps the family-wise
	// false-positive rate near 5% per response.
	var hits int
	for seed := range uint64(20) {
		x, trials := synthetic(3, 1, 100+seed)
		for i := range trials {
			trials[i].Res.AllocsPerOp = rand.New(rand.NewPCG(seed, uint64(i))).NormFloat64()
		}
		for _, e := range pairEffects(Analyze(x, trials), "allocs_op") {
			if e.Sig {
				hits++
			}
		}
	}
	if hits > 3 {
		t.Errorf("%d significant effects in 20 null experiments", hits)
	}
}

func TestFailedTrialsAreDroppedNotZeroed(t *testing.T) {
	x, trials := synthetic(3, 0.5, 3)
	trials[0].Res = failed(errInjected)
	eff := pairEffects(Analyze(x, trials), "allocs_op")
	if e := eff["X"]; math.Abs(e.Estimate-8) > 4*e.SE {
		t.Errorf("X = %.3f with one failed trial", e.Estimate)
	}
}
