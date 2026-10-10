// Command matrix compares two go test -json streams of the same module,
// run against two implementations, and prints a markdown matrix: every
// package with tests, then every test whose result differs, then the
// tests both runs skipped and why.
//
//	matrix -names v0.2.0,recon base.json recon.json
//
// It exits 1 when the second run fails a package or a test that the
// first one passes: the candidate must be a drop-in replacement.
package main

import (
	"cmp"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
)

func main() {
	names := flag.String("names", "baseline,candidate", "column names for the two runs")
	prefix := flag.String("trim", "", "import-path prefix to drop from package names")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: matrix [-names a,b] [-trim prefix] a.json b.json")
		os.Exit(2)
	}
	var runs [2]run
	for i, f := range flag.Args() {
		r, err := load(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, "matrix:", err)
			os.Exit(2)
		}
		runs[i] = r
	}
	col := strings.SplitN(*names+",", ",", 3)
	if regressions := render(os.Stdout, col[0], col[1], *prefix, runs[0], runs[1]); regressions > 0 {
		fmt.Fprintf(os.Stderr, "matrix: %d regressions in %s\n", regressions, col[1])
		os.Exit(1)
	}
}

func load(name string) (run, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parse(f)
}

// render writes the report and returns the number of regressions: a
// package or test that passes in a but not in b.
func render(w io.Writer, an, bn, trim string, a, b run) (regressions int) {
	all := slices.Sorted(maps.Keys(union(a, b)))
	short := func(p string) string { return cmp.Or(strings.TrimPrefix(p, trim), p) }

	fmt.Fprintf(w, "| package | %s | %s | tests pass/fail/skip (%s) | (%s) | seconds (%s) | (%s) |\n", an, bn, an, bn, an, bn)
	fmt.Fprintln(w, "|---|---|---|---:|---:|---:|---:|")
	noTests := 0
	for _, p := range all {
		pa, pb := a.get(p), b.get(p)
		if pa.result == "no-tests" && pb.result == "no-tests" {
			noTests++
			continue
		}
		if pa.result == "ok" && pb.result != "ok" {
			regressions++
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %.1f | %.1f |\n", short(p), mark(pa.result), mark(pb.result),
			countsOf(pa), countsOf(pb), pa.elapsed, pb.elapsed)
	}
	fmt.Fprintf(w, "\n%d packages have no test files.\n", noTests)

	fmt.Fprintf(w, "\n### Tests whose result differs\n\n")
	diffs := 0
	for _, p := range all {
		pa, pb := a.get(p), b.get(p)
		for _, t := range slices.Sorted(maps.Keys(union(pa.tests, pb.tests))) {
			ta, tb := deref(pa.tests[t]), deref(pb.tests[t])
			if ta.String() == tb.String() {
				continue
			}
			if diffs++; diffs == 1 {
				fmt.Fprintf(w, "| package | test | %s | %s |\n|---|---|---|---|\n", an, bn)
			}
			if passed(ta) && !passed(tb) {
				regressions++
			}
			fmt.Fprintf(w, "| %s | %s | %s | %s |\n", short(p), t, ta, tb)
		}
	}
	if diffs == 0 {
		fmt.Fprintln(w, "None: every test has the same result in both runs.")
	}

	fmt.Fprintf(w, "\n### Skipped in both runs\n\n")
	reasons := map[string][]string{}
	for _, p := range all {
		pa, pb := a.get(p), b.get(p)
		for t, ta := range pa.tests {
			if !passed(*ta) && ta.fail == 0 && deref(pb.tests[t]) == *ta {
				r := cmp.Or(logPrefix.ReplaceAllString(pa.why[t], ""), "(no reason logged)")
				reasons[r] = append(reasons[r], short(p)+"."+t)
			}
		}
	}
	if len(reasons) == 0 {
		fmt.Fprintln(w, "None.")
	}
	for _, r := range slices.Sorted(maps.Keys(reasons)) {
		ts := reasons[r]
		slices.Sort(ts)
		fmt.Fprintf(w, "- %d tests: `%s`\n  - %s\n", len(ts), r, strings.Join(ts, ", "))
	}
	return regressions
}

// logPrefix is t.Log's "file.go:12: " prefix; dropping it groups identical reasons.
var logPrefix = regexp.MustCompile(`^S+.go:d+: `)

func passed(t tally) bool { return t.pass > 0 && t.fail == 0 }

func countsOf(p *pkg) string {
	pass, fail, skip := p.counts()
	return fmt.Sprintf("%d/%d/%d", pass, fail, skip)
}

func mark(result string) string {
	if result == "ok" || result == "" {
		return cmp.Or(result, "absent")
	}
	return "**" + result + "**"
}

func (r run) get(p string) *pkg {
	if q := r[p]; q != nil {
		return q
	}
	return &pkg{result: "absent", tests: map[string]*tally{}, why: map[string]string{}}
}

func union[V any](a, b map[string]V) map[string]bool {
	u := map[string]bool{}
	for k := range a {
		u[k] = true
	}
	for k := range b {
		u[k] = true
	}
	return u
}

func deref(t *tally) tally {
	if t == nil {
		return tally{}
	}
	return *t
}
