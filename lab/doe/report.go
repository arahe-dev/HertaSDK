package doe

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Write stores an experiment's results in dir: runs.csv (one row per
// trial), effects.csv (every estimated effect), summary.json and REPORT.md.
func Write(dir string, x *Experiment, trials []Trial, an Analysis, meta Meta) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	sum := summarize(x, an, meta)
	js, err := marshalSummary(sum)
	if err != nil {
		return err
	}
	files := map[string][]byte{
		"runs.csv":     csvBytes(runRows(x, trials)),
		"effects.csv":  csvBytes(effectRows(an.Effects)),
		"summary.json": append(js, '\n'),
		"REPORT.md":    []byte(markdown(x, trials, an, sum)),
	}
	for name, b := range files {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func csvBytes(rows [][]string) []byte {
	var b strings.Builder
	w := csv.NewWriter(&b)
	w.WriteAll(rows) // a strings.Builder cannot fail
	return []byte(b.String())
}

func f64(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return ""
	}
	return strconv.FormatFloat(v, 'g', 6, 64)
}

func runRows(x *Experiment, trials []Trial) [][]string {
	head := []string{"seq", "rep", "point", "arm", "impl", "axis", "x"}
	head = append(head, Symbols(x.Factors)...)
	head = append(head, "clients", "rate", "capacity", "resources", "heavy_frac", "service_us",
		"admission", "keys", "zipf_s", "timeout_us", "fail_rate", "max_attempts", "cancel_rate",
		"took_ms", "err")
	for _, rs := range Responses {
		head = append(head, rs.Name)
	}
	rows := [][]string{append(head, "errors")}
	for _, t := range trials {
		p, w, arm := x.Points[t.Point], x.Points[t.Point].W, x.Arms[t.Arm]
		row := []string{strconv.Itoa(t.Seq), strconv.Itoa(t.Rep), strconv.Itoa(t.Point),
			arm.Label, arm.Impl.Name(), p.Axis, f64(p.X)}
		for _, l := range p.Levels {
			row = append(row, strconv.Itoa(int(l)))
		}
		row = append(row, strconv.Itoa(w.Clients), f64(w.Rate), strconv.FormatInt(w.Capacity, 10),
			strconv.Itoa(w.Resources), f64(w.HeavyFrac), f64(w.Service.Seconds()*1e6),
			[]string{"wait", "reject"}[w.Admission], strconv.Itoa(w.Keys), f64(w.ZipfS),
			f64(w.Timeout.Seconds()*1e6), f64(w.FailRate), strconv.Itoa(w.MaxAttempts),
			f64(w.CancelRate), f64(t.Took.Seconds()*1e3), t.Res.Err)
		for _, rs := range Responses {
			row = append(row, f64(rs.Value(&t.Res)))
		}
		rows = append(rows, append(row, errorList(t.Res.Errors)))
	}
	return rows
}

// errorList renders an error distribution as "class/outcome=n;…", sorted.
func errorList(m map[string]uint64) string {
	var parts []string
	for k, n := range m {
		parts = append(parts, fmt.Sprintf("%s=%d", k, n))
	}
	slices.Sort(parts)
	return strings.Join(parts, ";")
}

func effectRows(effects []Effect) [][]string {
	rows := [][]string{{"response", "scope", "effect", "order", "estimate", "se", "t", "df", "p",
		"ratio", "ci_lo", "ci_hi", "lenth", "significant"}}
	for _, e := range effects {
		lo, hi := e.CI()
		rows = append(rows, []string{e.Response, e.Scope, e.Name(), strconv.Itoa(e.Order),
			f64(e.Estimate), f64(e.SE), f64(e.T), f64(e.DF), f64(e.P), f64(e.Ratio()),
			f64(lo), f64(hi), e.Lenth, strconv.FormatBool(e.Sig)})
	}
	return rows
}

// topEffects returns, per scope, the n effects of one response with the
// largest |t| (|estimate| when t is undefined).
func topEffects(effects []Effect, resp string, n int) []Effect {
	var out []Effect
	for _, e := range effects {
		if e.Response == resp {
			out = append(out, e)
		}
	}
	slices.SortStableFunc(out, func(a, b Effect) int {
		if a.Scope != b.Scope {
			return strings.Compare(a.Scope, b.Scope)
		}
		return cmpAbs(strength(b), strength(a))
	})
	var kept []Effect
	count := map[string]int{}
	for _, e := range out {
		if count[e.Scope]++; count[e.Scope] <= n {
			kept = append(kept, e)
		}
	}
	return kept
}

func strength(e Effect) float64 {
	if math.IsNaN(e.T) {
		return 0
	}
	return e.T
}

func summarize(x *Experiment, an Analysis, meta Meta) Summary {
	s := Summary{Meta: meta, TopEffects: map[string][]effectJSON{}}
	if x.Kind != "sweep" && x.Design.K() > 0 {
		sym := Symbols(x.Factors)
		d := &designJSON{Runs: x.Design.Runs(), Resolution: x.Design.Resolution(),
			Generators: x.Design.Generators(sym)}
		for _, f := range x.Factors {
			d.Factors = append(d.Factors, factorJSON{f.Symbol, f.Name, f.Low, f.High})
		}
		for _, w := range x.Design.Defining() {
			d.Defining = append(d.Defining, w.Name(sym))
		}
		s.Design = d
	}
	for _, c := range an.Comparisons {
		cj := comparisonJSON{Response: c.Response, Unit: c.Unit, Base: c.Base, Cand: c.Cand,
			Verdict: c.Verdict, BaseMean: num(c.BaseMean), CandMean: num(c.CandMean),
			Effect: num(c.X.Estimate), SE: num(c.X.SE), P: num(c.X.P), Ratio: num(c.X.Ratio()),
			Lo: num(c.Lo), Hi: num(c.Hi)}
		for _, e := range c.Interactions {
			cj.Interactions = append(cj.Interactions, e.Name())
		}
		s.Comparisons = append(s.Comparisons, cj)
	}
	for _, n := range an.Noise {
		s.Noise = append(s.Noise, noiseJSON{n.Response, n.Scope, num(n.SD), num(n.CV), n.DF})
	}
	for _, rs := range Responses {
		for _, e := range topEffects(an.Effects, rs.Name, 5) {
			s.TopEffects[rs.Name] = append(s.TopEffects[rs.Name], effectJSON{e.Scope, e.Name(),
				num(e.Estimate), num(e.SE), num(e.P), num(e.Ratio()), e.Lenth, e.Sig})
		}
	}
	for _, c := range an.Curves {
		s.Curves = append(s.Curves, curveJSON{c.Axis, c.Arm, num(c.X), num(c.Throughput), num(c.P50),
			num(c.P99), num(c.P999), num(c.Allocs), num(c.PeakGoroutines), num(c.ErrRate)})
	}
	return s
}
