package mutate

import (
	"cmp"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
)

// WriteSummary renders a run as markdown: scores per operator and per
// stage, the tests that kill most, every survivor with its diff, and the
// equivalents with the reasons they cannot be killed.
func WriteSummary(w io.Writer, title string, stages []Stage, rs []Result, stale []string) {
	all, by := Tallies(rs)
	fmt.Fprintf(w, "# Mutation report: %s\n\n", title)
	fmt.Fprintf(w, "%d mutants. Score = killed / (killed + survived); equivalents are excluded, compile mutants are not counted. Raw counts equivalents as survivors.\n\n", all.Total)

	fmt.Fprintln(w, "| operator | mutants | compile | killed | survived | equivalent | score | raw |")
	fmt.Fprintln(w, "|---|---:|---:|---:|---:|---:|---:|---:|")
	row := func(name string, t Tally) {
		fmt.Fprintf(w, "| %s | %d | %d | %d | %d | %d | %s | %s |\n", name, t.Total, t.Compile,
			t.Killed, t.Survived, t.Equivalent, pct(t.Score()), pct(t.Raw()))
	}
	for _, op := range Operators {
		if t, ok := by[op]; ok {
			row(string(op), t)
		}
	}
	row("**all**", all)

	fmt.Fprint(w, "\n## Kill cascade\n\n")
	fmt.Fprintln(w, "| stage | command | kills | score if the cascade stopped here |")
	fmt.Fprintln(w, "|---|---|---:|---:|")
	var upTo []string
	for _, st := range stages {
		upTo = append(upTo, st.Name)
		fmt.Fprintf(w, "| %s | `%s` | %d | %s |\n", st.Name, strings.TrimPrefix(st.String(), st.Name+": "),
			all.ByStage[st.Name], pct(all.StageScore(upTo...)))
	}

	writeKillers(w, rs)
	writeFlaky(w, rs)

	sv := filter(rs, Survived)
	fmt.Fprintf(w, "\n## Survivors (%d)\n\n", len(sv))
	for _, r := range sv {
		fmt.Fprintf(w, "### #%d %s `%s` in %s\n\n`%s` → `%s`  \nkey: `%s`\n\n```diff\n%s```\n\n",
			r.ID, r.Where(), r.Op, r.Func, r.From, r.To, r.Key(), r.Diff)
	}
	eq := filter(rs, Equivalent)
	fmt.Fprintf(w, "## Equivalent (%d)\n\n", len(eq))
	for _, r := range eq {
		fmt.Fprintf(w, "- #%d %s `%s` `%s` → `%s`: %s\n", r.ID, r.Where(), r.Op, r.From, r.To, r.Reason)
	}
	if len(stale) > 0 {
		fmt.Fprintf(w, "\n## Stale triage entries (%d)\n\nThese verdicts matched no surviving mutant in this run; re-check them.\n\n", len(stale))
		for _, k := range stale {
			fmt.Fprintf(w, "- `%s`\n", k)
		}
	}
}

func writeKillers(w io.Writer, rs []Result) {
	n := map[string]int{}
	for _, r := range rs {
		for _, k := range r.Killers {
			n[r.Stage+" "+strings.SplitN(k, "/", 2)[0]]++
		}
	}
	keys := slices.SortedFunc(maps.Keys(n), func(a, b string) int {
		return cmp.Or(n[b]-n[a], strings.Compare(a, b))
	})
	fmt.Fprint(w, "\n## Tests that kill most\n\n")
	fmt.Fprintln(w, "| stage | test | mutants killed |")
	fmt.Fprintln(w, "|---|---|---:|")
	for _, k := range keys[:min(len(keys), 12)] {
		st, test, _ := strings.Cut(k, " ")
		fmt.Fprintf(w, "| %s | %s | %d |\n", st, test, n[k])
	}
}

// writeFlaky lists test failures that did not reproduce on a re-run of
// the same mutant: the tests' own flakiness under the run's load.
func writeFlaky(w io.Writer, rs []Result) {
	n := map[string]int{}
	for _, r := range rs {
		for _, f := range r.Flaky {
			n[f]++
		}
	}
	if len(n) == 0 {
		return
	}
	fmt.Fprint(w, "\n## Unreproduced failures\n\nThese tests failed once on a mutant and passed when re-run, so they killed nothing: they are flaky under the run's load.\n\n")
	fmt.Fprint(w, "| stage: test | mutants |\n|---|---:|\n")
	for _, k := range slices.Sorted(maps.Keys(n)) {
		fmt.Fprintf(w, "| %s | %d |\n", k, n[k])
	}
}

// WriteCompare renders several runs side by side, one column per run.
func WriteCompare(w io.Writer, names []string, runs [][]Result) {
	fmt.Fprintf(w, "| operator | %s |\n|---|%s\n", strings.Join(names, " | "), strings.Repeat("---:|", len(names)))
	tallies := make([]map[Operator]Tally, len(runs))
	totals := make([]Tally, len(runs))
	for i, rs := range runs {
		totals[i], tallies[i] = Tallies(rs)
	}
	cell := func(t Tally) string {
		return fmt.Sprintf("%s (%d/%d, %d eq)", pct(t.Score()), t.Killed, t.Killed+t.Survived, t.Equivalent)
	}
	for _, op := range Operators {
		cells := make([]string, len(runs))
		for i := range runs {
			cells[i] = cell(tallies[i][op])
		}
		fmt.Fprintf(w, "| %s | %s |\n", op, strings.Join(cells, " | "))
	}
	cells := make([]string, len(runs))
	for i := range runs {
		cells[i] = "**" + cell(totals[i]) + "**"
	}
	fmt.Fprintf(w, "| **all** | %s |\n", strings.Join(cells, " | "))
}

func filter(rs []Result, s Status) []Result {
	var out []Result
	for _, r := range rs {
		if r.Status == s {
			out = append(out, r)
		}
	}
	return out
}

func pct(f float64) string {
	if f < 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", 100*f)
}
