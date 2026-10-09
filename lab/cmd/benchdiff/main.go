// Command benchdiff compares two sets of `go test -bench` results, run by
// run: per benchmark and metric, the medians, a 95% bootstrap interval for
// their ratio, and the Mann-Whitney p-value. It is benchstat's core,
// stdlib only.
//
//	go run ./lab/cmd/benchdiff -base orig=a.txt -cand recon=b.txt [-csv out.csv]
//
// A delta is called significant when p < alpha and the ratio interval
// excludes 1.
package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"maps"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
)

// samples maps "benchmark-procs" → unit → one value per run.
type samples map[string]map[string][]float64

func main() {
	var base, cand, csvOut string
	var alpha float64
	flag.StringVar(&base, "base", "", "label=file of the baseline runs")
	flag.StringVar(&cand, "cand", "", "label=file of the candidate runs")
	flag.StringVar(&csvOut, "csv", "", "also write the table as CSV here")
	flag.Float64Var(&alpha, "alpha", 0.05, "significance level")
	flag.Parse()
	if err := run(base, cand, csvOut, alpha); err != nil {
		fmt.Fprintln(os.Stderr, "benchdiff:", err)
		os.Exit(1)
	}
}

func run(base, cand, csvOut string, alpha float64) error {
	bl, bs, err := load(base)
	if err != nil {
		return err
	}
	cl, cs, err := load(cand)
	if err != nil {
		return err
	}
	rows := compare(bs, cs, alpha)
	printMarkdown(os.Stdout, bl, cl, rows)
	if csvOut == "" {
		return nil
	}
	f, err := os.Create(csvOut)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	w.Write([]string{"benchmark", "unit", "n", bl + "_median", cl + "_median", "ratio", "ratio_lo", "ratio_hi", "p", "verdict"})
	for _, r := range rows {
		w.Write([]string{r.name, r.unit, strconv.Itoa(r.n), g(r.base), g(r.cand), g(r.ratio), g(r.lo), g(r.hi), g(r.p), r.verdict})
	}
	w.Flush()
	return errors.Join(w.Error(), f.Close())
}

func load(spec string) (string, samples, error) {
	label, path, ok := strings.Cut(spec, "=")
	if !ok {
		return "", nil, fmt.Errorf("want label=file, got %q", spec)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	s := samples{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parse(sc.Text(), s)
	}
	return label, s, sc.Err()
}

// parse folds one result line into s:
// "BenchmarkX-16  1000  123 ns/op  0 B/op  0 allocs/op". GOMAXPROCS=1
// carries no suffix; it is normalized to "-1".
func parse(line string, s samples) {
	f := strings.Fields(line)
	if len(f) < 4 || !strings.HasPrefix(f[0], "Benchmark") {
		return
	}
	name := strings.TrimPrefix(f[0], "Benchmark")
	if i := strings.LastIndexByte(name, '-'); i < 0 || !isDigits(name[i+1:]) {
		name += "-1"
	}
	if s[name] == nil {
		s[name] = map[string][]float64{}
	}
	for i := 2; i+1 < len(f); i += 2 {
		if v, err := strconv.ParseFloat(f[i], 64); err == nil {
			s[name][f[i+1]] = append(s[name][f[i+1]], v)
		}
	}
}

func isDigits(s string) bool {
	return s != "" && strings.Trim(s, "0123456789") == ""
}

type row struct {
	name, unit, verdict       string
	n                         int
	base, cand, ratio, lo, hi float64
	p                         float64
}

func compare(bs, cs samples, alpha float64) []row {
	var rows []row
	for _, name := range slices.Sorted(maps.Keys(bs)) {
		for _, unit := range []string{"ns/op", "B/op", "allocs/op"} {
			b, c := bs[name][unit], cs[name][unit]
			if len(b) == 0 || len(c) == 0 {
				continue
			}
			r := row{name: name, unit: unit, n: min(len(b), len(c)), base: median(b), cand: median(c), p: mannWhitney(b, c)}
			r.ratio = r.cand / r.base
			r.lo, r.hi = bootstrapRatio(b, c, 10000)
			r.verdict = verdict(r, alpha)
			rows = append(rows, r)
		}
	}
	return rows
}

func verdict(r row, alpha float64) string {
	switch {
	case r.base == r.cand:
		return "same"
	case r.p >= alpha || (r.lo <= 1 && r.hi >= 1):
		return "~"
	case r.cand < r.base:
		return "better"
	}
	return "worse"
}

func printMarkdown(w *os.File, bl, cl string, rows []row) {
	fmt.Fprintf(w, "| benchmark | unit | %s | %s | Δ | 95%% CI of Δ | p | verdict |\n|---|---|---:|---:|---:|---|---:|---|\n", bl, cl)
	for _, r := range rows {
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s | %s | %s |\n", r.name, r.unit,
			num(r.base), num(r.cand), pct(r.ratio), ci(r.lo, r.hi), pval(r.p), r.verdict)
	}
}

func num(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e6 {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

func pct(ratio float64) string {
	if math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return "n/a"
	}
	return fmt.Sprintf("%+.1f%%", 100*(ratio-1))
}

func ci(lo, hi float64) string {
	if math.IsNaN(lo) {
		return "n/a"
	}
	return fmt.Sprintf("[%s, %s]", pct(lo), pct(hi))
}

func pval(p float64) string {
	if p < 0.001 {
		return "<0.001"
	}
	return strconv.FormatFloat(p, 'f', 3, 64)
}

func g(v float64) string { return strconv.FormatFloat(v, 'g', 6, 64) }
