package doe

import (
	"fmt"
	"time"
)

// Standard screening designs over the ten Screening factors (A..H, J, K),
// both resolution IV: main effects are clear of two-factor interactions,
// and every X×factor interaction is clear of all two-factor interactions.
var (
	// Screen32 is the 2^(10-5) IV design, 32 runs (minimum aberration).
	Screen32 = mustFractional(10, "ABCD", "ABCE", "ABDE", "ACDE", "BCDE")
	// Screen64 is the 2^(10-4) IV design, 64 runs (minimum aberration),
	// with fewer aliased two-factor interactions than Screen32.
	Screen64 = mustFractional(10, "BCDF", "ACDF", "ABDE", "ABCE")
)

func mustFractional(k int, gens ...string) Design {
	d, err := Fractional(k, gens...)
	if err != nil {
		panic(err)
	}
	return d
}

// Screen is the screening experiment: the design, each run on every arm.
func Screen(d Design, base Workload, arms []Arm, reps int, seed uint64) *Experiment {
	return &Experiment{Kind: "screen", Factors: Screening, Design: d,
		Points: DesignPoints(d, Screening, base), Arms: arms, Reps: reps, Seed: seed}
}

// FullFactorial is the follow-up on a few factors (others at their low
// level): every interaction among them is estimable.
func FullFactorial(fs []Factor, base Workload, arms []Arm, reps int, seed uint64) (*Experiment, error) {
	if len(fs) == 0 || len(fs) > 6 {
		return nil, fmt.Errorf("doe: full factorial needs 1..6 factors, got %d", len(fs))
	}
	d := Full(len(fs))
	return &Experiment{Kind: "full", Factors: fs, Design: d,
		Points: DesignPoints(d, fs, base), Arms: arms, Reps: reps, Seed: seed}, nil
}

// Sweep is the 1-D curve experiment over the named axes.
func Sweep(axes []string, base Workload, arms []Arm, reps int, seed uint64) (*Experiment, error) {
	for _, a := range axes {
		if _, ok := sweepAxes[a]; !ok {
			return nil, fmt.Errorf("doe: unknown sweep axis %q (have %v)", a, SweepAxes)
		}
	}
	return &Experiment{Kind: "sweep", Points: SweepPoints(axes, base), Arms: arms, Reps: reps, Seed: seed}, nil
}

// AB is the single-workload comparison: one point, many replicates. With
// two labels on one implementation it is the A/A noise calibration.
func AB(base Workload, arms []Arm, reps int, seed uint64) *Experiment {
	return &Experiment{Kind: "ab", Design: Full(0), Points: []Point{{W: base}}, Arms: arms, Reps: reps, Seed: seed}
}

// ABWorkload resembles the production consumer: a contended pool, keyed
// writes with a hot key, a per-attempt timeout and rare transient failures.
func ABWorkload() Workload {
	w := Baseline()
	w.Clients, w.Capacity, w.Service = 64, 8, 50*time.Microsecond
	w.Keys, w.ZipfS = 256, 1.1
	w.Timeout, w.FailRate, w.MaxAttempts = 5*time.Millisecond, 0.01, 3
	return w
}
