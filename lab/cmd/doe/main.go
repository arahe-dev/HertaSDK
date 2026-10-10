// Command doe runs a Design-of-Experiments study of Herta implementations.
//
//	go run ./cmd/doe -design screen -impls orig,recon -reps 3 -dur 2s
//	go run ./cmd/doe -quick                       # whole screen in < 3 min
//	go run ./cmd/doe -full                        # 64-run screen, 5 reps, 3s windows
//	go run ./cmd/doe -design full -from results/<screen>/summary.json
//	go run ./cmd/doe -design sweep -sweep clients,capacity
//	go run ./cmd/doe -design ab -impls a=orig,b=orig   # A/A noise calibration
//
// -impls defaults to $LAB_IMPLS, else "orig,recon"; the first arm is the
// baseline. Results go to -out (default results/<run-id>/, inside lab/): runs.csv,
// effects.csv, summary.json and REPORT.md.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/arahe-dev/hertasdk/lab/doe"
)

type config struct {
	design, impls, out, factors, from, sweep string
	reps                                     int
	dur, warmup                              time.Duration
	seed                                     uint64
	quick, full, wide, quiet, spin           bool
}

func main() {
	var c config
	flag.StringVar(&c.design, "design", "screen", "screen | full | sweep | ab")
	flag.StringVar(&c.impls, "impls", doe.EnvArms(), `arms: "orig,recon" or "a=orig,b=orig"; first is the baseline`)
	flag.IntVar(&c.reps, "reps", 3, "replicates per (point, arm)")
	flag.DurationVar(&c.dur, "dur", 2*time.Second, "measurement window per trial")
	flag.DurationVar(&c.warmup, "warmup", 500*time.Millisecond, "warm-up before each window")
	flag.StringVar(&c.out, "out", "", "output directory (default results/<run-id>, relative to lab/)")
	flag.StringVar(&c.factors, "factors", "", "full: factors by symbol or name, e.g. A,B,keys")
	flag.StringVar(&c.from, "from", "", "full: pick the most significant factors from a screen's summary.json")
	flag.StringVar(&c.sweep, "sweep", strings.Join(doe.SweepAxes, ","), "sweep: axes to sweep")
	flag.Uint64Var(&c.seed, "seed", 1, "randomisation and workload seed")
	flag.BoolVar(&c.wide, "runs64", false, "screen: use the 64-run 2^(10-4) design instead of 32 runs")
	flag.BoolVar(&c.quick, "quick", false, "preset: 300ms windows, 100ms warm-up (screen < 3 min)")
	flag.BoolVar(&c.full, "full", false, "preset: 3s windows, 1s warm-up, 5 reps, 64-run screen")
	flag.BoolVar(&c.quiet, "q", false, "no per-trial progress")
	flag.BoolVar(&c.spin, "spin", false, "closed-loop callers retry at once after a refusal (default: yield first)")
	flag.Parse()
	if err := run(c); err != nil {
		fmt.Fprintln(os.Stderr, "doe:", err)
		os.Exit(1)
	}
}

// applyPresets sets the preset's values for every flag not given explicitly.
func applyPresets(c *config) {
	set := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { set[f.Name] = true })
	def := func(name string, apply func()) {
		if !set[name] {
			apply()
		}
	}
	ab := c.design == "ab" // one point: replicates are cheap and are all the power
	switch {
	case c.quick:
		def("dur", func() { c.dur = 300 * time.Millisecond })
		def("warmup", func() { c.warmup = 100 * time.Millisecond })
		def("reps", func() { c.reps = pick(ab, 8, 3) })
	case c.full:
		def("dur", func() { c.dur = 3 * time.Second })
		def("warmup", func() { c.warmup = time.Second })
		def("reps", func() { c.reps = pick(ab, 20, 5) })
		def("runs64", func() { c.wide = true })
	}
}

func run(c config) error {
	applyPresets(&c)
	arms, err := doe.ParseArms(c.impls)
	if err != nil {
		return err
	}
	x, err := experiment(c, arms)
	if err != nil {
		return err
	}
	start := time.Now()
	id := start.Format("20060102-150405") + "-" + c.design + "-" + armTag(arms)
	if c.out == "" {
		c.out = filepath.Join("results", id)
	}

	trials := x.Plan()
	fmt.Fprintf(os.Stderr, "doe %s: %d points × %d arms × %d reps = %d trials of %v+%v → %s\n",
		c.design, len(x.Points), len(arms), c.reps, len(trials), c.warmup, c.dur, c.out)
	x.Execute(trials, func(done int, t *doe.Trial) {
		if !c.quiet {
			progress(start, done, len(trials), x, t)
		}
	})
	an := doe.Analyze(x, trials)

	meta := doe.Meta{
		ID: id, Kind: c.design, Command: "go run ./cmd/doe " + strings.Join(os.Args[1:], " "),
		Started: start.Format(time.RFC3339), Go: runtime.Version(), OS: runtime.GOOS, Arch: runtime.GOARCH,
		CPUs: runtime.NumCPU(), GOMAXPROCS: runtime.GOMAXPROCS(0), Reps: c.reps, Seed: c.seed,
		WarmupMs: ms(c.warmup), DurationMs: ms(c.dur), Trials: len(trials),
		ElapsedS: time.Since(start).Seconds(),
	}
	for _, a := range arms {
		meta.Arms = append(meta.Arms, a.Label+"="+a.Impl.Name())
	}
	for _, t := range trials {
		if t.Res.Err != "" {
			meta.FailedTrials++
		}
	}
	if err := doe.Write(c.out, x, trials, an, meta); err != nil {
		return err
	}
	headline(an)
	fmt.Printf("\n%d trials (%d failed) in %.0fs → %s\n", len(trials), meta.FailedTrials, meta.ElapsedS,
		filepath.Join(c.out, "REPORT.md"))
	return nil
}

func experiment(c config, arms []doe.Arm) (*doe.Experiment, error) {
	window := func(w doe.Workload) doe.Workload {
		w.Warmup, w.Duration = c.warmup, c.dur
		w.SpinOnRefusal = c.spin
		return w
	}
	switch c.design {
	case "screen":
		d := doe.Screen32
		if c.wide {
			d = doe.Screen64
		}
		return doe.Screen(d, window(doe.Baseline()), arms, c.reps, c.seed), nil
	case "full":
		list := c.factors
		if c.from != "" {
			syms, err := factorsFrom(c.from)
			if err != nil {
				return nil, err
			}
			list = strings.Join(syms, ",")
			fmt.Fprintf(os.Stderr, "doe: factors from %s: %s\n", c.from, list)
		}
		fs, err := doe.SelectFactors(list)
		if err != nil {
			return nil, err
		}
		return doe.FullFactorial(fs, window(doe.Baseline()), arms, c.reps, c.seed)
	case "sweep":
		return doe.Sweep(strings.Split(c.sweep, ","), window(doe.SweepBase()), arms, c.reps, c.seed)
	case "ab":
		return doe.AB(window(doe.ABWorkload()), arms, c.reps, c.seed), nil
	}
	return nil, fmt.Errorf("unknown design %q", c.design)
}

func factorsFrom(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s doe.Summary
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	syms := doe.ActiveFactors(s, 5)
	if len(syms) == 0 {
		return nil, fmt.Errorf("%s: no significant factors", path)
	}
	return syms, nil
}

func progress(start time.Time, done, n int, x *doe.Experiment, t *doe.Trial) {
	el := time.Since(start)
	eta := time.Duration(float64(el) / float64(done) * float64(n-done))
	r := &t.Res
	status := fmt.Sprintf("%8.0f ops/s  p99 %8.1fµs  allocs/op %5.1f", r.Throughput, r.Lat.P99/1e3, r.AllocsPerOp)
	if r.Err != "" {
		status = "FAILED: " + r.Err
	}
	fmt.Fprintf(os.Stderr, "[%4d/%d eta %5s] rep %d pt %2d %-6s %s\n", done, n,
		eta.Round(time.Second), t.Rep, t.Point, x.Arms[t.Arm].Label, status)
}

// headline prints the key comparisons to stdout.
func headline(an doe.Analysis) {
	if len(an.Comparisons) == 0 {
		return
	}
	fmt.Printf("\n%-16s %-14s %12s %12s  %-8s %s\n", "response", "arms", "baseline", "candidate", "verdict", "p")
	for _, c := range an.Comparisons {
		if rs, _ := doe.ResponseByName(c.Response); rs.Key {
			fmt.Printf("%-16s %-14s %12.4g %12.4g  %-8s %.3g\n", c.Response, c.Base+"→"+c.Cand,
				c.BaseMean, c.CandMean, c.Verdict, c.X.P)
		}
	}
}

func armTag(arms []doe.Arm) string {
	var s []string
	for _, a := range arms {
		s = append(s, a.Label)
	}
	return strings.Join(s, "-")
}

func ms(d time.Duration) float64 { return float64(d) / 1e6 }

func pick(cond bool, yes, no int) int {
	if cond {
		return yes
	}
	return no
}
