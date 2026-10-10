package workload

import (
	"fmt"
	"slices"
	"time"
)

// Dist summarizes a latency sample.
type Dist struct {
	N                  int
	P50, P90, P99, Max time.Duration
}

// Summarize sorts ds in place and returns its quantiles (nearest rank).
func Summarize(ds []time.Duration) Dist {
	if len(ds) == 0 {
		return Dist{}
	}
	slices.Sort(ds)
	q := func(p float64) time.Duration { return ds[min(len(ds)-1, int(p*float64(len(ds))))] }
	return Dist{N: len(ds), P50: q(0.50), P90: q(0.90), P99: q(0.99), Max: ds[len(ds)-1]}
}

func (d Dist) String() string {
	return fmt.Sprintf("n=%d p50=%v p90=%v p99=%v max=%v", d.N, d.P50, d.P90, d.P99, d.Max)
}
