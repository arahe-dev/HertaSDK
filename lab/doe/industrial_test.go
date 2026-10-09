package doe

import (
	"os"
	"strings"
	"testing"
	"time"
)

// Industrial tier: LAB_LEVEL=full. These take minutes, not seconds.

func requireFull(t *testing.T) {
	if os.Getenv("LAB_LEVEL") != "full" {
		t.Skip("LAB_LEVEL=full only")
	}
}

// TestFullAACalibration runs the screening design with both arms on one
// implementation. Any Bonferroni-significant effect involving X is a lab
// false positive; the family-wise rate per response is 5%, so across all
// responses only a handful may appear by chance.
func TestFullAACalibration(t *testing.T) {
	requireFull(t)
	for _, im := range Impls(t) {
		t.Run(im.Name(), func(t *testing.T) {
			base := Baseline()
			base.Warmup, base.Duration = 50*time.Millisecond, 150*time.Millisecond
			x := Screen(Screen32, base, []Arm{{"a", im}, {"b", im}}, 2, 7)
			trials := x.Plan()
			x.Execute(trials, nil)
			var fp []string
			for _, e := range Analyze(x, trials).Effects {
				if strings.Contains(e.Scope, " vs ") && strings.Contains(e.Name(), ImplSymbol) && e.Sig {
					fp = append(fp, e.Response+":"+e.Name())
				}
			}
			t.Logf("%d false positives: %v", len(fp), fp)
			if len(fp) > 3 {
				t.Errorf("A/A produced %d significant X effects", len(fp))
			}
		})
	}
}

// TestFullSoak holds a production-shaped load with shutdown under load and
// checks the runtime's safety promises over a long window.
func TestFullSoak(t *testing.T) {
	requireFull(t)
	for _, im := range Impls(t) {
		t.Run(im.Name(), func(t *testing.T) {
			w := ABWorkload()
			w.Clients, w.CancelRate, w.HeavyFrac, w.Resources = 512, 0.02, 0.1, 2
			w.Warmup, w.Duration = time.Second, 20*time.Second
			r := Run(im, w)
			if r.Err != "" {
				t.Fatal(r.Err)
			}
			t.Logf("ops=%d tput=%.0f/s p99=%.0fµs p999=%.0fµs peakHeap=%.1fMiB peakG=%.0f drain=%.2fms errs=%v",
				r.Ops, r.Throughput, r.Lat.P99/1e3, r.Lat.P999/1e3, r.PeakHeapMB, r.PeakGoroutines, r.DrainMs, r.Errors)
			switch {
			case r.CapViolations+r.KeyViolations > 0:
				t.Errorf("violations: capacity %d key %d", r.CapViolations, r.KeyViolations)
			case r.DrainErr != "":
				t.Errorf("drain: %s", r.DrainErr)
			case r.Residual > 0:
				t.Errorf("%d goroutines left after shutdown", r.Residual)
			case r.PeakHeapMB > 256:
				t.Errorf("peak heap %.0f MiB", r.PeakHeapMB)
			}
		})
	}
}
