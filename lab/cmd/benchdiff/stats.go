package main

import (
	"math"
	"math/rand/v2"
	"slices"
)

func median(xs []float64) float64 {
	s := slices.Sorted(slices.Values(xs))
	m := len(s) / 2
	if len(s)%2 == 1 {
		return s[m]
	}
	return (s[m-1] + s[m]) / 2
}

// bootstrapRatio is a 95% percentile-bootstrap interval for
// median(cand)/median(base), resampling each sample independently.
// A zero base median makes the ratio undefined (NaN).
func bootstrapRatio(base, cand []float64, rounds int) (lo, hi float64) {
	rng := rand.New(rand.NewPCG(1, 2))
	ratios := make([]float64, 0, rounds)
	rb, rc := make([]float64, len(base)), make([]float64, len(cand))
	for range rounds {
		for i := range rb {
			rb[i] = base[rng.IntN(len(base))]
		}
		for i := range rc {
			rc[i] = cand[rng.IntN(len(cand))]
		}
		if mb := median(rb); mb != 0 {
			ratios = append(ratios, median(rc)/mb)
		}
	}
	if len(ratios) < rounds/2 {
		return math.NaN(), math.NaN()
	}
	slices.Sort(ratios)
	return ratios[int(0.025*float64(len(ratios)))], ratios[int(0.975*float64(len(ratios)))-1]
}

// mannWhitney is the two-sided p-value of the Mann-Whitney U test, by the
// normal approximation with tie and continuity corrections. Two samples
// made of one shared value give p = 1.
func mannWhitney(a, b []float64) float64 {
	type obs struct {
		v float64
		a bool
	}
	all := make([]obs, 0, len(a)+len(b))
	for _, v := range a {
		all = append(all, obs{v, true})
	}
	for _, v := range b {
		all = append(all, obs{v, false})
	}
	slices.SortFunc(all, func(x, y obs) int {
		switch {
		case x.v < y.v:
			return -1
		case x.v > y.v:
			return 1
		}
		return 0
	})
	n1, n2 := float64(len(a)), float64(len(b))
	n := n1 + n2
	var rankA, ties float64
	for i := 0; i < len(all); {
		j := i
		for j < len(all) && all[j].v == all[i].v {
			j++
		}
		rank := float64(i+j+1) / 2 // mean of ranks i+1 .. j
		for k := i; k < j; k++ {
			if all[k].a {
				rankA += rank
			}
		}
		t := float64(j - i)
		ties += t*t*t - t
		i = j
	}
	u := rankA - n1*(n1+1)/2
	sigma := math.Sqrt(n1 * n2 / 12 * ((n + 1) - ties/(n*(n-1))))
	if sigma == 0 {
		return 1
	}
	z := (math.Abs(u-n1*n2/2) - 0.5) / sigma
	return math.Min(1, math.Erfc(max(z, 0)/math.Sqrt2))
}
