package main

import (
	"math"
	"testing"
)

func TestMannWhitney(t *testing.T) {
	lo := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	hi := []float64{11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	// Complete separation of 10 vs 10: exact p = 2/C(20,10) ≈ 1.1e-5; the
	// normal approximation gives ≈ 1.8e-4.
	if p := mannWhitney(lo, hi); p > 3e-4 {
		t.Errorf("separated samples: p = %g", p)
	}
	if p := mannWhitney(lo, lo); p < 0.9 {
		t.Errorf("identical samples: p = %g", p)
	}
	if p := mannWhitney([]float64{0, 0, 0}, []float64{0, 0, 0}); p != 1 {
		t.Errorf("all ties: p = %g", p)
	}
	if p := mannWhitney([]float64{0, 0, 0, 0, 0}, []float64{10, 10, 10, 10, 10}); p > 0.02 {
		t.Errorf("constant but different: p = %g", p)
	}
}

func TestBootstrapRatio(t *testing.T) {
	base := []float64{100, 101, 99, 100, 102, 98, 100, 101, 99, 100}
	cand := []float64{50, 51, 49, 50, 50, 50, 51, 49, 50, 50}
	lo, hi := bootstrapRatio(base, cand, 2000)
	if !(lo <= 0.5 && 0.5 <= hi && hi < 0.53 && lo > 0.47) {
		t.Errorf("ratio CI [%g, %g] should hug 0.5", lo, hi)
	}
	if lo, _ := bootstrapRatio([]float64{0, 0}, cand, 100); !math.IsNaN(lo) {
		t.Errorf("zero base: want NaN, got %g", lo)
	}
}

func TestParse(t *testing.T) {
	s := samples{}
	parse("BenchmarkDoUncontended   \t 9000000\t  52.1 ns/op\t 0 B/op\t 0 allocs/op", s)
	parse("BenchmarkRetryOverhead/retry/max_attempts=3-16  100  700 ns/op  80 B/op  2 allocs/op", s)
	if got := s["DoUncontended-1"]["ns/op"]; len(got) != 1 || got[0] != 52.1 {
		t.Errorf("GOMAXPROCS=1 line: %v", s)
	}
	if got := s["RetryOverhead/retry/max_attempts=3-16"]["allocs/op"]; len(got) != 1 || got[0] != 2 {
		t.Errorf("sub-benchmark line: %v", s)
	}
}
