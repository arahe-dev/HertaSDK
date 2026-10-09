package doe

import (
	"math"
	"math/bits"
	"sync/atomic"
)

// Hist is a log-bucketed latency histogram in the HDR style: values below
// 2^histSub are exact, larger values fall in one of 2^histSub linear
// sub-buckets per power of two, so the bucket width is at most 1/64 of the
// value and the reported midpoint is within 0.8% of any value in it.
//
// Record is lock-free and allocation-free; many goroutines may record into
// one Hist. Reads (Quantile, Merge) are meant for after recording stops.
type Hist struct {
	counts [histBuckets]uint64
	n      uint64
	sum    uint64
	max    uint64
}

const (
	histSub     = 6  // log2(sub-buckets per octave)
	histMaxExp  = 40 // values ≥ 2^40 ns (~18 min) clamp
	histBuckets = (histMaxExp - histSub + 2) << histSub
)

func histIndex(v uint64) int {
	if v < 1<<histSub {
		return int(v)
	}
	e := bits.Len64(v) - 1
	if e > histMaxExp {
		return histBuckets - 1
	}
	sub := (v >> (e - histSub)) & (1<<histSub - 1)
	return (e-histSub+1)<<histSub | int(sub)
}

// histBounds returns the half-open value range [lo, hi) of bucket i.
func histBounds(i int) (lo, hi uint64) {
	if i < 1<<histSub {
		return uint64(i), uint64(i) + 1
	}
	e := i>>histSub + histSub - 1
	sub := uint64(i & (1<<histSub - 1))
	lo = (1<<histSub + sub) << (e - histSub)
	return lo, lo + 1<<(e-histSub)
}

// Record adds one value (nanoseconds; negatives count as 0).
func (h *Hist) Record(ns int64) {
	v := uint64(max(ns, 0))
	atomic.AddUint64(&h.counts[histIndex(v)], 1)
	atomic.AddUint64(&h.n, 1)
	atomic.AddUint64(&h.sum, v)
	for {
		m := atomic.LoadUint64(&h.max)
		if v <= m || atomic.CompareAndSwapUint64(&h.max, m, v) {
			return
		}
	}
}

// Merge adds o's contents to h.
func (h *Hist) Merge(o *Hist) {
	for i, c := range o.counts {
		h.counts[i] += c
	}
	h.n += o.n
	h.sum += o.sum
	h.max = max(h.max, o.max)
}

// Count is the number of recorded values.
func (h *Hist) Count() uint64 { return h.n }

// Max is the exact largest recorded value.
func (h *Hist) Max() float64 { return float64(h.max) }

// Mean is the exact arithmetic mean, or NaN when empty.
func (h *Hist) Mean() float64 {
	if h.n == 0 {
		return math.NaN()
	}
	return float64(h.sum) / float64(h.n)
}

// Quantile returns the q-quantile (0 < q ≤ 1) as the midpoint of the bucket
// holding the ceil(q·n)-th smallest value, capped at Max. NaN when empty.
func (h *Hist) Quantile(q float64) float64 {
	if h.n == 0 {
		return math.NaN()
	}
	rank := uint64(math.Ceil(q * float64(h.n)))
	rank = min(max(rank, 1), h.n)
	var seen uint64
	for i, c := range h.counts {
		if seen += c; seen >= rank {
			lo, hi := histBounds(i)
			if hi-lo == 1 {
				return float64(lo)
			}
			return math.Min(float64(lo+hi-1)/2, float64(h.max))
		}
	}
	return float64(h.max)
}
