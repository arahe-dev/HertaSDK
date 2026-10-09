package doe

import (
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// TestRunSmoke drives every selected implementation through a closed and an
// open loop and checks the engine's own invariants: calls complete, the
// runtime keeps its capacity and key promises, and shutdown drains.
func TestRunSmoke(t *testing.T) {
	cases := map[string]func(*Workload){
		"closed": func(w *Workload) { w.Clients, w.Keys, w.ZipfS, w.Service = 32, 8, 1.2, 20*time.Microsecond },
		"open":   func(w *Workload) { w.Clients, w.Rate, w.Service = 64, 20000, 20*time.Microsecond },
		"mixed": func(w *Workload) {
			w.Clients, w.Resources, w.HeavyFrac, w.Capacity = 16, 3, 0.25, 8
			w.FailRate, w.MaxAttempts, w.Timeout, w.CancelRate = 0.1, 3, time.Millisecond, 0.05
		},
	}
	for _, im := range Impls(t) {
		for name, set := range cases {
			t.Run(im.Name()+"/"+name, func(t *testing.T) {
				w := Baseline()
				w.Warmup, w.Duration = 20*time.Millisecond, 100*time.Millisecond
				set(&w)
				r := Run(im, w)
				if r.Err != "" {
					t.Skipf("%s cannot run: %s", im.Name(), r.Err)
				}
				t.Logf("ops=%d ok=%d tput=%.0f/s p50=%.0fns p99=%.0fns allocs/op=%.1f peakG=%.0f drain=%.2fms errs=%v",
					r.Ops, r.OK, r.Throughput, r.Lat.P50, r.Lat.P99, r.AllocsPerOp, r.PeakGoroutines, r.DrainMs, r.Errors)
				if r.Ops == 0 {
					t.Fatal("no calls completed")
				}
				if r.CapViolations+r.KeyViolations > 0 {
					t.Errorf("violations: capacity %d, key %d", r.CapViolations, r.KeyViolations)
				}
				if r.DrainErr != "" {
					t.Errorf("drain: %s", r.DrainErr)
				}
			})
		}
	}
}

func Impls(t testing.TB) []impl.Impl {
	arms, err := ParseArms(EnvArms())
	if err != nil {
		t.Fatal(err)
	}
	var out []impl.Impl
	for _, a := range arms {
		out = append(out, a.Impl)
	}
	return out
}
