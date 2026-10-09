package doe

import "math"

// Result is everything measured in one run. Rates and quantiles cover only
// calls that started and finished inside the measurement window.
type Result struct {
	Err string `json:",omitempty"` // the run could not execute; responses are NaN

	Seconds             float64 // measurement window
	Ops, OK             uint64  // completed Do calls; of which nil error
	Throughput, Goodput float64 // Ops and OK per second

	Lat, Wait, Exec Quantiles // Do latency (ok calls); admission wait; execution

	AllocsPerOp, BytesPerOp float64 // process-wide MemStats deltas / Ops
	GCPerSec, GCPauseUs     float64
	PeakGoroutines          float64 // beyond the harness's own goroutines
	PeakHeapMB              float64

	Errors       map[string]uint64 // "class/outcome" → count
	RetriesPerOp float64
	Jain         float64 // fairness over per-caller completions (closed loop)
	DropRate     float64 // open loop: arrivals dropped because the queue was full

	CapViolations, KeyViolations uint64

	// Shutdown under load.
	DrainMs  float64
	DrainErr string `json:",omitempty"`
	StatsLag uint64 // Admitted - Finished observed right after Shutdown returned nil
	Hung     bool   // callers still blocked 5s after Shutdown
	Residual int    // goroutines left 50ms after the run, beyond the baseline
}

// Quantiles summarises one histogram, in nanoseconds.
type Quantiles struct {
	P50, P90, P99, P999, Max, Mean float64
}

func quantiles(h *Hist) Quantiles {
	return Quantiles{
		P50: h.Quantile(0.5), P90: h.Quantile(0.9), P99: h.Quantile(0.99),
		P999: h.Quantile(0.999), Max: h.Max(), Mean: h.Mean(),
	}
}

func (r *Result) ErrRate() float64 {
	if r.Ops == 0 {
		return math.NaN()
	}
	return float64(r.Ops-r.OK) / float64(r.Ops)
}
