package doe

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"
)

func TestHistQuantilesWithinOnePercent(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	dists := map[string]func() int64{
		"uniform":     func() int64 { return rng.Int64N(1_000_000) },
		"exponential": func() int64 { return int64(rng.ExpFloat64() * 50_000) },
		"lognormal":   func() int64 { return int64(math.Exp(rng.NormFloat64()*2 + 8)) },
	}
	for name, draw := range dists {
		var h Hist
		xs := make([]int64, 200_000)
		for i := range xs {
			xs[i] = draw()
			h.Record(xs[i])
		}
		slices.Sort(xs)
		for _, q := range []float64{0.5, 0.9, 0.99, 0.999} {
			exact := float64(xs[int(math.Ceil(q*float64(len(xs))))-1])
			got := h.Quantile(q)
			if tol := math.Max(0.01*exact, 1); math.Abs(got-exact) > tol {
				t.Errorf("%s q%.3f: got %.0f, exact %.0f", name, q, got, exact)
			}
		}
		if h.Max() != float64(xs[len(xs)-1]) {
			t.Errorf("%s max %v != %d", name, h.Max(), xs[len(xs)-1])
		}
	}
}

func TestHistBucketsTileTheLine(t *testing.T) {
	for i := range histBuckets - 1 {
		lo, hi := histBounds(i)
		if nlo, _ := histBounds(i + 1); nlo != hi {
			t.Fatalf("gap after bucket %d: [%d,%d) then %d", i, lo, hi, nlo)
		}
		if histIndex(lo) != i || histIndex(hi-1) != i {
			t.Fatalf("bucket %d [%d,%d) maps to %d..%d", i, lo, hi, histIndex(lo), histIndex(hi-1))
		}
		if lo >= 1<<histSub && float64(hi-lo)/float64(lo) > 1.0/(1<<histSub)+1e-12 {
			t.Fatalf("bucket %d too wide", i)
		}
	}
}

func TestHistSmallValuesExactAndClamp(t *testing.T) {
	var h Hist
	for v := range int64(64) {
		h.Record(v)
	}
	if q := h.Quantile(0.5); q != 31 {
		t.Errorf("median of 0..63 = %v", q)
	}
	h.Record(-5)
	h.Record(math.MaxInt64)
	if h.Count() != 66 || h.Max() != math.MaxInt64 {
		t.Errorf("count %d max %v", h.Count(), h.Max())
	}
}

func TestHistRecordDoesNotAllocate(t *testing.T) {
	var h Hist
	if a := testing.AllocsPerRun(1000, func() { h.Record(12345) }); a != 0 {
		t.Errorf("Record allocates %v", a)
	}
}

func TestHistMerge(t *testing.T) {
	var a, b Hist
	a.Record(100)
	b.Record(300)
	a.Merge(&b)
	if a.Count() != 2 || a.Mean() != 200 || a.Max() != 300 {
		t.Errorf("merged: n=%d mean=%v max=%v", a.Count(), a.Mean(), a.Max())
	}
}
