package doe

import (
	"math/rand/v2"
	"runtime"
	"time"
)

// Experiment is a set of workload points, each run on every arm, Reps
// times. Within each replicate (a block) the point × arm order is shuffled,
// so slow drift of the machine spreads over all treatments instead of
// biasing one.
type Experiment struct {
	Kind    string // screen | full | sweep | ab
	Factors []Factor
	Design  Design // over Factors; zero for sweeps
	Points  []Point
	Arms    []Arm
	Reps    int
	Seed    uint64
}

// Point is one workload of an experiment: a design run (Levels) or a sweep
// value (X along Axis).
type Point struct {
	Levels []int8
	Axis   string
	X      float64
	W      Workload
}

// Trial is one execution: a point on an arm in a replicate.
type Trial struct {
	Seq, Rep, Point, Arm int
	Res                  Result
	Took                 time.Duration
}

// DesignPoints applies each design run's levels to base.
func DesignPoints(d Design, fs []Factor, base Workload) []Point {
	pts := make([]Point, d.Runs())
	for r := range pts {
		w := base
		lv := make([]int8, len(fs))
		for f := range fs {
			lv[f] = d.Level(r, f)
			fs[f].Set(&w, lv[f] > 0)
		}
		pts[r] = Point{Levels: lv, W: w}
	}
	return pts
}

// Plan lists every trial in execution order.
func (x *Experiment) Plan() []Trial {
	rng := rand.New(rand.NewPCG(x.Seed, 0x9e3779b97f4a7c15))
	var runs []Trial
	for rep := range x.Reps {
		block := make([]Trial, 0, len(x.Points)*len(x.Arms))
		for p := range x.Points {
			for a := range x.Arms {
				block = append(block, Trial{Rep: rep, Point: p, Arm: a})
			}
		}
		rng.Shuffle(len(block), func(i, j int) { block[i], block[j] = block[j], block[i] })
		runs = append(runs, block...)
	}
	for i := range runs {
		runs[i].Seq = i
	}
	return runs
}

// Execute performs the runs in order, each from a collected heap, calling
// progress after each. Every run gets its own seed derived from the
// experiment seed, the replicate and the point — never the arm, so arms
// see identical arrival and key sequences (common random numbers).
func (x *Experiment) Execute(runs []Trial, progress func(done int, r *Trial)) {
	for i := range runs {
		r := &runs[i]
		w := x.Points[r.Point].W
		w.Seed = x.Seed ^ uint64(r.Rep)<<32 ^ uint64(r.Point)
		runtime.GC()
		t := time.Now()
		r.Res = Run(x.Arms[r.Arm].Impl, w)
		r.Took = time.Since(t)
		if progress != nil {
			progress(i+1, r)
		}
	}
}
