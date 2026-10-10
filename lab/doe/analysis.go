package doe

import (
	"math"
	"strings"
)

// Effect is one estimated contrast (alias class) of one response.
type Effect struct {
	Response string
	Scope    string   // an arm label, or "base→cand" for the crossed pair
	Aliases  []string // the alias class, shortest first (orders ≤ 2 listed)
	Order    int      // order of the shortest alias
	Log      bool     // Estimate is in log2 units

	Estimate, SE, T, P, DF float64
	Lenth                  string // "*" beyond Lenth's ME, "**" beyond SME
	Sig                    bool   // P < 0.05/m, m = contrasts estimated (Bonferroni)
}

// Name is the alias chain, e.g. "AB=CF".
func (e Effect) Name() string { return strings.Join(e.Aliases, "=") }

// Ratio is the multiplicative effect 2^Estimate of a log response (the
// +1 level over the -1 level); NaN for raw responses.
func (e Effect) Ratio() float64 {
	if !e.Log {
		return math.NaN()
	}
	return math.Exp2(e.Estimate)
}

// CI is the 95% confidence interval of Estimate.
func (e Effect) CI() (lo, hi float64) {
	h := studentQuantile(0.975, e.DF) * e.SE
	return e.Estimate - h, e.Estimate + h
}

// HasFactor reports whether the effect's shortest alias involves symbol s.
func (e Effect) HasFactor(s string) bool { return strings.Contains(e.Aliases[0], s) }

// Noise is the replicate (pure) error of one response in one scope.
type Noise struct {
	Response, Scope string
	SD, DF          float64 // pooled within-cell SD, analysis scale
	CV              float64 // log: 2^SD-1; raw: SD/|grand mean|
}

// obs is one trial's response on the analysis scale, in design cell c.
type obs struct {
	cell int
	y    float64
}

// estimate fits every contrast of design d to the observations. Contrasts
// use cell means, so unequal replication stays unbiased; the standard error
// comes from the pooled within-cell variance. It returns nil if any cell
// has no valid observation.
func estimate(d Design, symbols []string, rs Response, scope string, data []obs) ([]Effect, Noise) {
	nc := d.Runs()
	sum, n := make([]float64, nc), make([]float64, nc)
	for _, o := range data {
		if !math.IsNaN(o.y) && !math.IsInf(o.y, 0) {
			sum[o.cell] += o.y
			n[o.cell]++
		}
	}
	mean := make([]float64, nc)
	var grand, invN, total float64
	for c := range nc {
		if n[c] == 0 {
			return nil, Noise{}
		}
		mean[c] = sum[c] / n[c]
		grand += mean[c] / float64(nc)
		invN += 1 / n[c]
		total += n[c]
	}
	var ss float64
	for _, o := range data {
		if !math.IsNaN(o.y) && !math.IsInf(o.y, 0) {
			ss += (o.y - mean[o.cell]) * (o.y - mean[o.cell])
		}
	}
	df := total - float64(nc)
	s2 := ss / df
	k := 2 / float64(nc)
	se := k * math.Sqrt(s2*invN)
	noise := Noise{Response: rs.Name, Scope: scope, SD: math.Sqrt(s2), DF: df}
	noise.CV = noise.SD / math.Abs(grand)
	if rs.Log {
		noise.CV = math.Exp2(noise.SD) - 1
	}

	classes := d.Contrasts()
	effects := make([]Effect, len(classes))
	ests := make([]float64, len(classes))
	for i, cl := range classes {
		var est float64
		for c := range nc {
			est += float64(d.Sign(c, cl[0])) * mean[c]
		}
		est *= k
		ests[i] = est
		t := est / se
		effects[i] = Effect{
			Response: rs.Name, Scope: scope, Aliases: aliasNames(cl, symbols),
			Order: cl[0].Order(), Log: rs.Log,
			Estimate: est, SE: se, T: t, P: twoSidedP(t, df), DF: df,
		}
		effects[i].Sig = effects[i].P < 0.05/float64(len(classes))
	}
	if _, me, sme := lenth(ests); !math.IsNaN(me) {
		for i := range effects {
			switch a := math.Abs(ests[i]); {
			case a > sme && sme > 0:
				effects[i].Lenth = "**"
			case a > me && me > 0:
				effects[i].Lenth = "*"
			}
		}
	}
	return effects, noise
}

func aliasNames(cl []Word, symbols []string) []string {
	out := []string{cl[0].Name(symbols)}
	for _, w := range cl[1:] {
		if w.Order() <= 2 {
			out = append(out, w.Name(symbols))
		}
	}
	return out
}
