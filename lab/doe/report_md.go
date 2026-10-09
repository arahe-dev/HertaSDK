package doe

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
)

// markdown renders REPORT.md.
func markdown(x *Experiment, trials []Trial, an Analysis, s Summary) string {
	var b strings.Builder
	m := s.Meta
	fmt.Fprintf(&b, "# DoE %s: %s\n\n", m.Kind, strings.Join(m.Arms, " vs "))
	fmt.Fprintf(&b, "`%s` · %s · %s %s/%s · %d CPUs (GOMAXPROCS %d) · %d trials (%d failed) in %.0fs · %d reps · warm-up %.0fms + window %.0fms · seed %d\n\n",
		m.ID, m.Started, m.Go, m.OS, m.Arch, m.CPUs, m.GOMAXPROCS, m.Trials, m.FailedTrials,
		m.ElapsedS, m.Reps, m.WarmupMs, m.DurationMs, m.Seed)
	fmt.Fprintf(&b, "Command: `%s`\n\n", m.Command)
	if sameImpl(x.Arms) {
		b.WriteString("> **A/A calibration.** Every arm runs the same implementation, so every significant X effect below is a false positive of the lab, and the X confidence intervals measure the smallest difference this setup can resolve.\n\n")
	}
	writeComparisons(&b, an)
	writeNoise(&b, x, an)
	writeErrors(&b, x, trials)
	writeDesign(&b, x)
	writeEffects(&b, x, an)
	writePoints(&b, x, trials)
	writeCurves(&b, an)
	writeFailures(&b, x, trials)
	return b.String()
}

func sameImpl(arms []Arm) bool {
	return len(arms) > 1 && !slices.ContainsFunc(arms, func(a Arm) bool { return a.Impl != arms[0].Impl })
}

// table writes a GitHub-flavoured Markdown table.
func table(b *strings.Builder, head []string, rows [][]string) {
	b.WriteString("| " + strings.Join(head, " | ") + " |\n|")
	for range head {
		b.WriteString("---|")
	}
	b.WriteString("\n")
	for _, r := range rows {
		b.WriteString("| " + strings.Join(r, " | ") + " |\n")
	}
	b.WriteString("\n")
}

// writeComparisons is the headline: every candidate arm against the baseline.
func writeComparisons(b *strings.Builder, an Analysis) {
	if len(an.Comparisons) == 0 {
		return
	}
	b.WriteString("## Verdict\n\n")
	b.WriteString("X is the implementation factor (−1 baseline, +1 candidate), crossed with every design run, so its main effect is the candidate's change averaged over the whole design. Log-scale responses show the ratio candidate/baseline of geometric means; raw responses show the difference. Significance is Bonferroni-corrected over all contrasts of a response; ratios within 5% count as *same*.\n\n")
	for _, key := range []bool{true, false} {
		var rows [][]string
		for _, c := range an.Comparisons {
			rs := mustResponse(c.Response)
			if rs.Key != key {
				continue
			}
			inter := "–"
			if len(c.Interactions) > 0 {
				e := c.Interactions[0]
				inter = fmt.Sprintf("%s (%s)", e.Name(), effectText(e))
			}
			v := c.Verdict
			if v == "better" || v == "worse" {
				v = "**" + v + "**"
			}
			rows = append(rows, []string{c.Response, c.Base + "→" + c.Cand, human(c.BaseMean, rs.Unit),
				human(c.CandMean, rs.Unit), change(c, rs.Log), pval(c.X.P), v, inter})
		}
		if key {
			b.WriteString("### Key responses\n\n")
		} else {
			b.WriteString("<details><summary>All other responses</summary>\n\n")
		}
		table(b, []string{"response", "arms", "baseline", "candidate", "change [95% CI]", "p", "verdict", "strongest X×factor"}, rows)
		if !key {
			b.WriteString("</details>\n\n")
		}
	}
}

func effectText(e Effect) string {
	if e.Log {
		return fmt.Sprintf("×%.3g", e.Ratio())
	}
	return fmt.Sprintf("%+.3g", e.Estimate)
}

// writeNoise reports the replicate error of a single trial, per arm.
func writeNoise(b *strings.Builder, x *Experiment, an Analysis) {
	if x.Reps < 2 || len(an.Noise) == 0 {
		return
	}
	b.WriteString("## Noise\n\nReplicate error of a single trial (pooled within-cell SD) as a coefficient of variation: the ±1σ spread between identical trials.\n\n")
	head := []string{"response"}
	for _, a := range x.Arms {
		head = append(head, a.Label)
	}
	var rows [][]string
	for _, rs := range Responses {
		row := []string{rs.Name}
		for _, a := range x.Arms {
			cv := "–"
			for _, n := range an.Noise {
				if n.Response == rs.Name && n.Scope == a.Label && !math.IsNaN(n.CV) && !math.IsInf(n.CV, 0) {
					cv = fmt.Sprintf("%.1f%%", 100*n.CV)
				}
			}
			row = append(row, cv)
		}
		rows = append(rows, row)
	}
	table(b, head, rows)
}

// writeFailures lists trials that could not run.
func writeFailures(b *strings.Builder, x *Experiment, trials []Trial) {
	var rows [][]string
	for _, t := range trials {
		if t.Res.Err != "" {
			rows = append(rows, []string{fmt.Sprint(t.Seq), x.Arms[t.Arm].Label, fmt.Sprint(t.Point), t.Res.Err})
		}
	}
	if len(rows) == 0 {
		return
	}
	fmt.Fprintf(b, "## Failed trials (%d)\n\nFailed trials are excluded from estimation; a design cell with none left is not estimated.\n\n", len(rows))
	if len(rows) > 20 {
		rows = append(rows[:20], []string{"…", "", "", fmt.Sprintf("%d more in runs.csv", len(rows)-20)})
	}
	table(b, []string{"seq", "arm", "point", "error"}, rows)
}

// writeErrors tabulates every error kind ("class/outcome") per arm, as a
// share of all calls in all of the arm's trials.
func writeErrors(b *strings.Builder, x *Experiment, trials []Trial) {
	counts := make([]map[string]uint64, len(x.Arms))
	ops := make([]uint64, len(x.Arms))
	kinds := map[string]bool{}
	for i := range counts {
		counts[i] = map[string]uint64{}
	}
	for _, t := range trials {
		ops[t.Arm] += t.Res.Ops
		for k, n := range t.Res.Errors {
			counts[t.Arm][k] += n
			kinds[k] = true
		}
	}
	if len(kinds) == 0 {
		return
	}
	b.WriteString("## Error classes\n\nShare of all calls, over all trials of each arm. A class is the implementation-neutral error class, the outcome what `OutcomeOf` reports.\n\n")
	head := []string{"class/outcome"}
	for _, a := range x.Arms {
		head = append(head, a.Label)
	}
	var rows [][]string
	for _, k := range slices.Sorted(maps.Keys(kinds)) {
		row := []string{k}
		for a := range x.Arms {
			row = append(row, fmt.Sprintf("%.3f%%", 100*float64(counts[a][k])/float64(max(ops[a], 1))))
		}
		rows = append(rows, row)
	}
	table(b, head, rows)
}
