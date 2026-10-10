package doe

import (
	"math"
	"time"
)

// Sweep axes: one workload knob varied on a log2 grid, everything else held
// at SweepBase.
var sweepAxes = map[string]struct {
	values []float64
	set    func(w *Workload, x float64)
}{
	"clients":  {pow2(0, 10), func(w *Workload, x float64) { w.Clients = int(x) }},
	"capacity": {pow2(0, 6), func(w *Workload, x float64) { w.Capacity = int64(x) }},
}

// SweepAxes lists the sweepable axes in report order.
var SweepAxes = []string{"clients", "capacity"}

// SweepBase is the workload sweeps perturb: a contended budget with a short
// service time, where admission cost and queueing both show.
func SweepBase() Workload {
	w := Baseline()
	w.Clients, w.Capacity, w.Service = 64, 16, 50*time.Microsecond
	return w
}

func pow2(lo, hi int) []float64 {
	var out []float64
	for e := lo; e <= hi; e++ {
		out = append(out, math.Exp2(float64(e)))
	}
	return out
}

// SweepPoints builds the points of the named axes on top of base.
func SweepPoints(axes []string, base Workload) []Point {
	var pts []Point
	for _, a := range axes {
		ax := sweepAxes[a]
		for _, x := range ax.values {
			w := base
			ax.set(&w, x)
			pts = append(pts, Point{Axis: a, X: x, W: w})
		}
	}
	return pts
}

// CurvePoint is one sweep point on one arm: medians over replicates.
type CurvePoint struct {
	Axis, Arm                  string
	X                          float64
	Throughput, P50, P99, P999 float64
	Allocs, PeakGoroutines     float64
	ErrRate                    float64
}

func curves(x *Experiment, trials []Trial) []CurvePoint {
	var out []CurvePoint
	for p, pt := range x.Points {
		for a, arm := range x.Arms {
			var rs []*Result
			for i := range trials {
				if trials[i].Point == p && trials[i].Arm == a {
					rs = append(rs, &trials[i].Res)
				}
			}
			med := func(get func(*Result) float64) float64 {
				var ys []float64
				for _, r := range rs {
					if r.Err == "" {
						ys = append(ys, get(r))
					}
				}
				return median(ys)
			}
			out = append(out, CurvePoint{
				Axis: pt.Axis, Arm: arm.Label, X: pt.X,
				Throughput:     med(func(r *Result) float64 { return r.Throughput }),
				P50:            med(func(r *Result) float64 { return r.Lat.P50 }),
				P99:            med(func(r *Result) float64 { return r.Lat.P99 }),
				P999:           med(func(r *Result) float64 { return r.Lat.P999 }),
				Allocs:         med(func(r *Result) float64 { return r.AllocsPerOp }),
				PeakGoroutines: med(func(r *Result) float64 { return r.PeakGoroutines }),
				ErrRate:        med((*Result).ErrRate),
			})
		}
	}
	return out
}
