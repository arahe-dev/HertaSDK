package doe

import "math"

// Response is one measured quantity of a run and how it is analysed.
//
// Log responses are analysed as log2(max(y, Floor)): effects are then
// multiplicative and read as ratios (2^effect), which is how latency and
// throughput behave. Raw responses (counts, fractions, allocations) are
// analysed in their own units, where a difference is the meaningful
// quantity and zero is a legitimate value.
type Response struct {
	Name   string
	Unit   string
	Log    bool
	Floor  float64
	Better int  // -1 lower is better, +1 higher is better, 0 neutral
	Key    bool // in the headline comparison
	Get    func(*Result) float64
}

// Value is the response on its natural scale (NaN for a failed run).
func (rs Response) Value(r *Result) float64 {
	if r.Err != "" {
		return math.NaN()
	}
	return rs.Get(r)
}

// Scaled is the response on its analysis scale.
func (rs Response) Scaled(r *Result) float64 {
	v := rs.Value(r)
	if rs.Log && !math.IsNaN(v) {
		return math.Log2(max(v, rs.Floor))
	}
	return v
}

func u(x uint64) float64 { return float64(x) }

// Responses is the fixed response catalog, in report order.
var Responses = []Response{
	{"throughput", "ops/s", true, 1, +1, true, func(r *Result) float64 { return r.Throughput }},
	// goodput is a key response: under Reject a faster runtime turns
	// closed-loop clients around sooner, so throughput and err_rate both
	// rise with the rejection rate; successful calls per second do not.
	{"goodput", "ops/s", true, 1, +1, true, func(r *Result) float64 { return r.Goodput }},
	{"lat_p50", "ns", true, 10, -1, true, func(r *Result) float64 { return r.Lat.P50 }},
	{"lat_p90", "ns", true, 10, -1, false, func(r *Result) float64 { return r.Lat.P90 }},
	{"lat_p99", "ns", true, 10, -1, true, func(r *Result) float64 { return r.Lat.P99 }},
	{"lat_p999", "ns", true, 10, -1, true, func(r *Result) float64 { return r.Lat.P999 }},
	{"lat_max", "ns", true, 10, -1, false, func(r *Result) float64 { return r.Lat.Max }},
	{"wait_p50", "ns", true, 10, -1, false, func(r *Result) float64 { return r.Wait.P50 }},
	{"wait_p99", "ns", true, 10, -1, true, func(r *Result) float64 { return r.Wait.P99 }},
	{"exec_p50", "ns", true, 10, -1, false, func(r *Result) float64 { return r.Exec.P50 }},
	{"exec_p99", "ns", true, 10, -1, false, func(r *Result) float64 { return r.Exec.P99 }},
	{"allocs_op", "allocs", false, 0, -1, true, func(r *Result) float64 { return r.AllocsPerOp }},
	{"bytes_op", "B", false, 0, -1, true, func(r *Result) float64 { return r.BytesPerOp }},
	{"peak_goroutines", "goroutines", false, 0, -1, true, func(r *Result) float64 { return r.PeakGoroutines }},
	{"peak_heap", "MiB", true, 0.01, -1, false, func(r *Result) float64 { return r.PeakHeapMB }},
	{"gc_per_s", "GC/s", false, 0, -1, false, func(r *Result) float64 { return r.GCPerSec }},
	{"err_rate", "fraction", false, 0, -1, true, func(r *Result) float64 { return r.ErrRate() }},
	{"retries_op", "retries", false, 0, 0, false, func(r *Result) float64 { return r.RetriesPerOp }},
	{"fairness", "jain", false, 0, +1, false, func(r *Result) float64 { return r.Jain }},
	{"cap_violations", "count", false, 0, -1, true, func(r *Result) float64 { return u(r.CapViolations) }},
	{"key_violations", "count", false, 0, -1, true, func(r *Result) float64 { return u(r.KeyViolations) }},
	{"drain", "ms", true, 0.001, -1, true, func(r *Result) float64 { return r.DrainMs }},
	{"stats_lag", "calls", false, 0, -1, false, func(r *Result) float64 { return u(r.StatsLag) }},
	{"residual_goroutines", "goroutines", false, 0, -1, false, func(r *Result) float64 { return float64(r.Residual) }},
}

// ResponseByName returns the named response, or false.
func ResponseByName(n string) (Response, bool) {
	for _, r := range Responses {
		if r.Name == n {
			return r, true
		}
	}
	return Response{}, false
}
