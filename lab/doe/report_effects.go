package doe

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

// writeDesign documents the design: factors, generators, aliasing, matrix.
func writeDesign(b *strings.Builder, x *Experiment) {
	if x.Kind == "sweep" || x.Design.K() == 0 {
		return
	}
	d, sym := x.Design, Symbols(x.Factors)
	fmt.Fprintf(b, "## Design\n\n%d runs × %d arms × %d reps, randomised within each replicate block. ", d.Runs(), len(x.Arms), x.Reps)
	if res := d.Resolution(); res > 0 {
		fmt.Fprintf(b, "2^(%d−%d) fractional factorial, resolution %s; generators %s.\n\n",
			d.K(), d.K()-d.Base, roman(res), strings.Join(d.Generators(sym), ", "))
	} else {
		fmt.Fprintf(b, "2^%d full factorial: every interaction is estimable.\n\n", d.K())
	}
	var rows [][]string
	for _, f := range x.Factors {
		rows = append(rows, []string{f.Symbol, f.Name, f.Low, f.High})
	}
	table(b, []string{"symbol", "factor", "low (−1)", "high (+1)"}, rows)
	if def := d.Defining(); len(def) > 0 {
		var words []string
		for _, w := range def {
			if w.Order() <= 4 {
				words = append(words, w.Name(sym))
			}
		}
		fmt.Fprintf(b, "Defining relation: I = %s (+ %d longer words, listed in summary.json)\n\n",
			strings.Join(words, " = "), len(def)-len(words))
		var chains []string
		for _, cl := range d.Contrasts() {
			if cl[0].Order() == 2 && len(cl) > 1 && cl[1].Order() == 2 {
				chains = append(chains, strings.Join(aliasNames(cl, sym), "="))
			}
		}
		if len(chains) > 0 {
			fmt.Fprintf(b, "Aliased two-factor interactions (each chain is one estimate): %s. Interactions with X are clear of every two-factor interaction.\n\n", strings.Join(chains, ", "))
		}
	}
	b.WriteString("<details><summary>Design matrix</summary>\n\n```\nrun " + strings.Join(sym, " ") + "\n")
	for r := range d.Runs() {
		fmt.Fprintf(b, "%3d %s\n", r, levelString(x.Points[r].Levels, " "))
	}
	b.WriteString("```\n\n</details>\n\n")
}

func roman(n int) string {
	return [...]string{"0", "I", "II", "III", "IV", "V", "VI", "VII", "VIII"}[min(n, 8)]
}

func levelString(ls []int8, sep string) string {
	s := make([]string, len(ls))
	for i, l := range ls {
		s[i] = "-"
		if l > 0 {
			s[i] = "+"
		}
	}
	return strings.Join(s, sep)
}

// writeEffects ranks the effects of each key response, per scope.
func writeEffects(b *strings.Builder, x *Experiment, an Analysis) {
	if len(an.Effects) == 0 || x.Design.K() == 0 {
		return
	}
	b.WriteString("## Effects\n\nTop effects per scope by |t|. An effect is the change from the low to the high level (a ratio for log-scale responses). `*`/`**`: beyond Lenth's ME/SME; **sig**: Bonferroni p < 0.05/m on replicate error. effects.csv has every contrast.\n\n")
	for _, rs := range Responses {
		if !rs.Key {
			continue
		}
		top := topEffects(an.Effects, rs.Name, 6)
		if len(top) == 0 {
			continue
		}
		fmt.Fprintf(b, "### %s (%s)\n\n", rs.Name, rs.Unit)
		var rows [][]string
		for _, e := range top {
			sig := ""
			if e.Sig {
				sig = "**sig**"
			}
			rows = append(rows, []string{e.Scope, e.Name(), effectText(e), ciText(e), pval(e.P), e.Lenth, sig})
		}
		table(b, []string{"scope", "effect", "estimate", "95% CI", "p", "Lenth", ""}, rows)
	}
	for _, name := range []string{"lat_p99", "throughput"} {
		writeHalfNormal(b, x, an, name)
	}
}

// writeHalfNormal prints the half-normal plot of the crossed scope's
// contrasts: |effect| against half-normal quantiles. Inert effects lie on a
// line through the origin; active ones stand off it.
func writeHalfNormal(b *strings.Builder, x *Experiment, an Analysis, resp string) {
	if len(x.Arms) < 2 {
		return
	}
	scope := x.Arms[0].Label + " vs " + x.Arms[1].Label
	var es []Effect
	for _, e := range an.Effects {
		if e.Response == resp && e.Scope == scope {
			es = append(es, e)
		}
	}
	if len(es) < 3 {
		return
	}
	slices.SortFunc(es, func(a, b Effect) int { return cmpAbs(a.Estimate, b.Estimate) })
	top := math.Abs(es[len(es)-1].Estimate)
	fmt.Fprintf(b, "<details><summary>Half-normal plot: %s, %s (%d contrasts)</summary>\n\n```\n rank  z(half)  |effect|\n", resp, scope, len(es))
	for i := len(es) - 1; i >= 0; i-- {
		e := es[i]
		z := normalQuantile(0.5 + 0.5*(float64(i)+0.5)/float64(len(es)))
		fmt.Fprintf(b, "%5d  %6.2f  %8.3f  %-30s %s %s\n", len(es)-i, z, math.Abs(e.Estimate),
			bar(math.Abs(e.Estimate)/top, 30), e.Name(), e.Lenth)
	}
	b.WriteString("```\n\n</details>\n\n")
}

// ciText is an effect's 95% confidence interval on its natural scale.
func ciText(e Effect) string {
	lo, hi := e.CI()
	if math.IsNaN(lo) {
		return "–"
	}
	if e.Log {
		return fmt.Sprintf("[%.3g, %.3g]", math.Exp2(lo), math.Exp2(hi))
	}
	return fmt.Sprintf("[%+.3g, %+.3g]", lo, hi)
}
