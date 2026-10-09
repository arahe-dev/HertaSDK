package doe

import (
	"fmt"
	"math"
	"strings"
)

// human renders v in unit compactly: "1.23µs", "84.1k", "0.031".
func human(v float64, unit string) string {
	switch {
	case math.IsNaN(v):
		return "–"
	case unit == "ns":
		return humanNs(v)
	case unit == "ms":
		return humanNs(v * 1e6)
	case unit == "fraction":
		return fmt.Sprintf("%.2f%%", 100*v)
	}
	a := math.Abs(v)
	switch {
	case a >= 1e9:
		return fmt.Sprintf("%.3gG", v/1e9)
	case a >= 1e6:
		return fmt.Sprintf("%.3gM", v/1e6)
	case a >= 1e4:
		return fmt.Sprintf("%.3gk", v/1e3)
	case a >= 100 || v == math.Trunc(v):
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.3g", v)
}

func humanNs(ns float64) string {
	switch a := math.Abs(ns); {
	case a >= 1e9:
		return fmt.Sprintf("%.3gs", ns/1e9)
	case a >= 1e6:
		return fmt.Sprintf("%.3gms", ns/1e6)
	case a >= 1e3:
		return fmt.Sprintf("%.3gµs", ns/1e3)
	}
	return fmt.Sprintf("%.0fns", ns)
}

// pval renders a p-value for a table.
func pval(p float64) string {
	switch {
	case math.IsNaN(p):
		return "–"
	case p < 1e-4:
		return "<1e-4"
	}
	return fmt.Sprintf("%.3g", p)
}

// change renders a comparison's effect: a ratio with CI for log responses,
// a signed difference with CI for raw ones.
func change(c Comparison, log bool) string {
	if c.X.Aliases == nil {
		return "–"
	}
	if log {
		return fmt.Sprintf("×%.3g [%.3g, %.3g]", c.X.Ratio(), c.Lo, c.Hi)
	}
	return fmt.Sprintf("%+.3g [%+.3g, %+.3g]", c.X.Estimate, c.Lo, c.Hi)
}

// bar is a horizontal bar of frac ∈ [0,1] over width cells, in eighths.
func bar(frac float64, width int) string {
	if math.IsNaN(frac) || frac <= 0 {
		return ""
	}
	eighths := int(math.Round(math.Min(frac, 1) * float64(width*8)))
	partial := []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
	return strings.Repeat("█", eighths/8) + partial[eighths%8]
}
