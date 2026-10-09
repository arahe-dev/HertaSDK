package doe

import (
	"math"
	"slices"
)

// Analysis is everything derived from an experiment's trials.
type Analysis struct {
	Effects     []Effect     // per arm, and per crossed (baseline, candidate) pair
	Noise       []Noise      // replicate error per response and scope
	Comparisons []Comparison // the implementation factor, per response
	Curves      []CurvePoint // sweeps only
}

// Comparison is the headline answer for one response: how the candidate
// arm differs from the baseline, averaged over the whole design.
type Comparison struct {
	Response, Unit     string
	Base, Cand         string
	BaseMean, CandMean float64  // geometric means for log responses
	X                  Effect   // the implementation main effect
	Lo, Hi             float64  // 95% CI: of the ratio (log) or difference (raw)
	Verdict            string   // better | worse | same | differs | n/a
	Interactions       []Effect // significant X×factor effects, strongest first
}

// materialLog is the smallest ratio change a verdict calls a difference.
const materialLog = 0.0704 // log2(1.05): 5%

// Analyze estimates every effect of every response in every scope.
func Analyze(x *Experiment, trials []Trial) Analysis {
	var an Analysis
	if x.Kind == "sweep" {
		an.Curves = curves(x, trials)
		return an
	}
	symbols := Symbols(x.Factors)
	crossed, xsyms := x.Design.Cross(), append(slices.Clip(symbols), ImplSymbol)
	for _, rs := range Responses {
		for a, arm := range x.Arms {
			var data []obs
			for _, t := range trials {
				if t.Arm == a {
					data = append(data, obs{t.Point, rs.Scaled(&t.Res)})
				}
			}
			eff, noise := estimate(x.Design, symbols, rs, arm.Label, data)
			an.Effects = append(an.Effects, eff...)
			an.Noise = append(an.Noise, noise)
		}
		for j := 1; j < len(x.Arms); j++ {
			scope := x.Arms[0].Label + " vs " + x.Arms[j].Label
			var data []obs
			for _, t := range trials {
				if t.Arm == 0 || t.Arm == j {
					cell := t.Point
					if t.Arm == j {
						cell |= 1 << x.Design.Base
					}
					data = append(data, obs{cell, rs.Scaled(&t.Res)})
				}
			}
			eff, _ := estimate(crossed, xsyms, rs, scope, data)
			an.Effects = append(an.Effects, eff...)
			an.Comparisons = append(an.Comparisons, compare(rs, x.Arms[0], x.Arms[j], eff, trials, 0, j))
		}
	}
	return an
}

func compare(rs Response, base, cand Arm, eff []Effect, trials []Trial, b, c int) Comparison {
	cmp := Comparison{Response: rs.Name, Unit: rs.Unit, Base: base.Label, Cand: cand.Label,
		BaseMean: armMean(rs, trials, b), CandMean: armMean(rs, trials, c), Verdict: "n/a"}
	for _, e := range eff {
		switch {
		case e.Aliases[0] == ImplSymbol:
			cmp.X = e
		case e.HasFactor(ImplSymbol) && e.Order == 2 && e.Sig:
			cmp.Interactions = append(cmp.Interactions, e)
		}
	}
	slices.SortFunc(cmp.Interactions, func(a, b Effect) int { return cmpAbs(b.T, a.T) })
	if cmp.X.Aliases == nil {
		cmp.Lo, cmp.Hi = math.NaN(), math.NaN()
		return cmp
	}
	cmp.Lo, cmp.Hi = cmp.X.CI()
	if rs.Log {
		cmp.Lo, cmp.Hi = math.Exp2(cmp.Lo), math.Exp2(cmp.Hi)
	}
	cmp.Verdict = verdict(rs, cmp.X)
	return cmp
}

func verdict(rs Response, x Effect) string {
	switch {
	case math.IsNaN(x.P) && x.Estimate == 0:
		return "same" // identical, noise-free (e.g. zero violations on both)
	case math.IsNaN(x.P):
		return "n/a"
	case !x.Sig, rs.Log && math.Abs(x.Estimate) < materialLog:
		return "same"
	case rs.Better == 0:
		return "differs"
	case (x.Estimate > 0) == (rs.Better > 0):
		return "better"
	}
	return "worse"
}

// armMean is the arm's mean over all its trials: geometric for log
// responses (the mean on the analysis scale, transformed back).
func armMean(rs Response, trials []Trial, arm int) float64 {
	var ys []float64
	for _, t := range trials {
		if y := rs.Scaled(&t.Res); t.Arm == arm && !math.IsNaN(y) {
			ys = append(ys, y)
		}
	}
	m, _ := meanVar(ys)
	if rs.Log {
		return math.Exp2(m)
	}
	return m
}

func cmpAbs(a, b float64) int {
	a, b = math.Abs(a), math.Abs(b)
	switch {
	case a < b || math.IsNaN(a):
		return -1
	case a > b:
		return 1
	}
	return 0
}
