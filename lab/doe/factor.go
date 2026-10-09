package doe

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// Factor is a two-level experimental factor: Set moves a workload to its
// low (-1) or high (+1) level.
type Factor struct {
	Symbol    string // one letter, used in effect names ("AB" = A×B)
	Name      string
	Low, High string // human-readable levels
	Set       func(w *Workload, high bool)
}

// ImplSymbol names the implementation factor when arms are crossed with a
// design: -1 is the baseline arm, +1 the candidate.
const ImplSymbol = "X"

// Screening is the catalog of screening factors, in base-first order for
// the standard designs. Symbol I is skipped (it denotes the identity).
var Screening = []Factor{
	{"A", "clients", "8", "256", func(w *Workload, h bool) { w.Clients = pick(h, 8, 256) }},
	{"B", "capacity", "4", "16", func(w *Workload, h bool) { w.Capacity = pick[int64](h, 4, 16) }},
	{"C", "service", "0", "100µs", func(w *Workload, h bool) { w.Service = pick(h, 0, 100*time.Microsecond) }},
	{"D", "admission", "wait", "reject", func(w *Workload, h bool) { w.Admission = pick(h, impl.Wait, impl.Reject) }},
	{"E", "keys", "none", "64·zipf1.2", func(w *Workload, h bool) { w.Keys, w.ZipfS = pick(h, 0, 64), pick(h, 0, 1.2) }},
	{"F", "timeout", "none", "1ms", func(w *Workload, h bool) { w.Timeout = pick(h, 0, time.Millisecond) }},
	{"G", "failures", "0%·1try", "10%·3tries", func(w *Workload, h bool) {
		w.FailRate, w.MaxAttempts = pick(h, 0, 0.1), pick(h, 1, 3)
	}},
	{"H", "weights", "all 1u", "25% cap/2", func(w *Workload, h bool) { w.HeavyFrac = pick(h, 0, 0.25) }},
	{"J", "resources", "1", "3", func(w *Workload, h bool) { w.Resources = pick(h, 1, 3) }},
	{"K", "cancel", "0%", "5%", func(w *Workload, h bool) { w.CancelRate = pick(h, 0, 0.05) }},
}

func pick[T any](high bool, lo, hi T) T {
	if high {
		return hi
	}
	return lo
}

// Symbols returns the factors' symbols, in order.
func Symbols(fs []Factor) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Symbol
	}
	return out
}

// SelectFactors resolves a comma list of symbols or names ("A,keys") against
// the screening catalog, keeping catalog order and symbols.
func SelectFactors(list string) ([]Factor, error) {
	want := map[string]bool{}
	for _, s := range strings.Split(list, ",") {
		if s = strings.TrimSpace(s); s != "" {
			want[strings.ToLower(s)] = true
		}
	}
	var out []Factor
	for _, f := range Screening {
		if want[strings.ToLower(f.Symbol)] || want[f.Name] {
			delete(want, strings.ToLower(f.Symbol))
			delete(want, f.Name)
			out = append(out, f)
		}
	}
	if len(want) > 0 {
		return nil, fmt.Errorf("doe: unknown factors %v", slices.Sorted(maps.Keys(want)))
	}
	return out, nil
}
