package doe

import (
	"math"
	"slices"
)

// meanVar returns the sample mean and unbiased variance (NaN variance for n<2).
func meanVar(xs []float64) (mean, variance float64) {
	if len(xs) == 0 {
		return math.NaN(), math.NaN()
	}
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	if len(xs) < 2 {
		return mean, math.NaN()
	}
	for _, x := range xs {
		variance += (x - mean) * (x - mean)
	}
	return mean, variance / float64(len(xs)-1)
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	s := slices.Sorted(slices.Values(xs))
	m := len(s) / 2
	if len(s)%2 == 1 {
		return s[m]
	}
	return (s[m-1] + s[m]) / 2
}

// jain is Jain's fairness index (Σx)²/(n·Σx²): 1 when all equal, 1/n when
// one client takes everything. NaN for no data.
func jain(xs []float64) float64 {
	var s, s2 float64
	for _, x := range xs {
		s += x
		s2 += x * x
	}
	if s2 == 0 {
		return math.NaN()
	}
	return s * s / (float64(len(xs)) * s2)
}

// normalQuantile is the standard normal inverse CDF.
func normalQuantile(p float64) float64 { return math.Sqrt2 * math.Erfinv(2*p-1) }

// studentCDF is P(T ≤ t) for Student's t with df degrees of freedom.
func studentCDF(t, df float64) float64 {
	if math.IsInf(df, 1) {
		return 0.5 * math.Erfc(-t/math.Sqrt2)
	}
	tail := 0.5 * betaInc(df/2, 0.5, df/(df+t*t))
	if t > 0 {
		return 1 - tail
	}
	return tail
}

// studentQuantile inverts studentCDF by bisection (|t| < 1e3 is plenty).
func studentQuantile(p, df float64) float64 {
	lo, hi := -1e3, 1e3
	for range 200 {
		mid := (lo + hi) / 2
		if studentCDF(mid, df) < p {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// twoSidedP is the two-sided p-value of a t statistic.
func twoSidedP(t, df float64) float64 {
	if math.IsNaN(t) || df <= 0 {
		return math.NaN()
	}
	return 2 * studentCDF(-math.Abs(t), df)
}

// betaInc is the regularized incomplete beta I_x(a, b), evaluated with the
// Lentz continued fraction on whichever side converges fast.
func betaInc(a, b, x float64) float64 {
	switch {
	case x <= 0:
		return 0
	case x >= 1:
		return 1
	}
	la, _ := math.Lgamma(a)
	lb, _ := math.Lgamma(b)
	lab, _ := math.Lgamma(a + b)
	front := math.Exp(lab - la - lb + a*math.Log(x) + b*math.Log1p(-x))
	if x < (a+1)/(a+b+2) {
		return front * betaCF(a, b, x) / a
	}
	return 1 - front*betaCF(b, a, 1-x)/b
}

func betaCF(a, b, x float64) float64 {
	const tiny, eps = 1e-300, 1e-15
	clamp := func(v float64) float64 {
		if math.Abs(v) < tiny {
			return tiny
		}
		return v
	}
	c, d := 1.0, 1/clamp(1-(a+b)*x/(a+1))
	f := d
	for m := 1; m <= 300; m++ {
		fm := float64(m)
		for _, num := range [2]float64{
			fm * (b - fm) * x / ((a + 2*fm - 1) * (a + 2*fm)),
			-(a + fm) * (a + b + fm) * x / ((a + 2*fm) * (a + 2*fm + 1)),
		} {
			d = 1 / clamp(1+num*d)
			c = clamp(1 + num/c)
			f *= c * d
		}
		if math.Abs(c*d-1) < eps {
			break
		}
	}
	return f
}

// lenth computes Lenth's pseudo standard error of a set of effect estimates
// from an unreplicated (or cell-mean) 2-level design, plus the margin of
// error ME and simultaneous margin SME at 95%. Effects beyond ME are
// individually active; beyond SME, active with family-wise control.
func lenth(effects []float64) (pse, me, sme float64) {
	m := len(effects)
	if m < 3 {
		return math.NaN(), math.NaN(), math.NaN()
	}
	abs := make([]float64, 0, m)
	for _, e := range effects {
		abs = append(abs, math.Abs(e))
	}
	s0 := 1.5 * median(abs)
	trimmed := abs[:0:0]
	for _, a := range abs {
		if a < 2.5*s0 {
			trimmed = append(trimmed, a)
		}
	}
	pse = 1.5 * median(trimmed)
	d := float64(m) / 3
	gamma := (1 + math.Pow(0.95, 1/float64(m))) / 2
	return pse, studentQuantile(0.975, d) * pse, studentQuantile(gamma, d) * pse
}
