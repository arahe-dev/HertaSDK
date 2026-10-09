package doe

import (
	"math"
	"testing"
)

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestStudentQuantileMatchesTables(t *testing.T) {
	for _, c := range []struct{ p, df, want float64 }{
		{0.975, 1, 12.706}, {0.975, 5, 2.571}, {0.975, 10, 2.228},
		{0.975, 30, 2.042}, {0.995, 20, 2.845}, {0.95, 3, 2.353},
		{0.975, math.Inf(1), 1.960},
	} {
		if got := studentQuantile(c.p, c.df); !near(got, c.want, 0.002) {
			t.Errorf("t(%v, %v) = %.4f, want %.3f", c.p, c.df, got, c.want)
		}
	}
	if p := twoSidedP(2.228, 10); !near(p, 0.05, 0.0005) {
		t.Errorf("p(2.228, 10) = %.5f", p)
	}
}

func TestNormalQuantile(t *testing.T) {
	if q := normalQuantile(0.975); !near(q, 1.95996, 1e-4) {
		t.Errorf("z(.975) = %v", q)
	}
}

func TestJain(t *testing.T) {
	if j := jain([]float64{5, 5, 5, 5}); !near(j, 1, 1e-12) {
		t.Errorf("equal shares: %v", j)
	}
	if j := jain([]float64{8, 0, 0, 0}); !near(j, 0.25, 1e-12) {
		t.Errorf("one hog: %v", j)
	}
}

func TestLenthFlagsOnlyLargeEffects(t *testing.T) {
	// 15 effects: 12 noise-sized, 3 large.
	eff := []float64{0.1, -0.2, 0.15, -0.05, 0.12, -0.18, 0.08, 0.02, -0.11, 0.07, -0.09, 0.14, 5, -4, 3}
	pse, me, sme := lenth(eff)
	if pse <= 0 || me <= pse || sme <= me {
		t.Fatalf("pse=%v me=%v sme=%v", pse, me, sme)
	}
	for i, e := range eff {
		if active := math.Abs(e) > me; active != (i >= 12) {
			t.Errorf("effect %v: active=%v (ME %.3f)", e, active, me)
		}
	}
}

func TestMeanVar(t *testing.T) {
	m, v := meanVar([]float64{2, 4, 4, 4, 5, 5, 7, 9})
	if m != 5 || !near(v, 32.0/7, 1e-12) {
		t.Errorf("mean %v var %v", m, v)
	}
}
