package doe

import (
	"cmp"
	"slices"
	"strings"
)

// ActiveFactors picks the follow-up factors from a screen: the screening
// symbols that appear most often in significant effects (main effects and
// interactions, X excluded) of the key responses, at most n, in catalog
// order so the follow-up design reads like the screen.
func ActiveFactors(s Summary, n int) []string {
	score := map[string]int{}
	for resp, effects := range s.TopEffects {
		if rs, ok := ResponseByName(resp); !ok || !rs.Key {
			continue
		}
		for _, e := range effects {
			if !e.Significant {
				continue
			}
			shortest, _, _ := strings.Cut(e.Effect, "=")
			for _, r := range shortest {
				if sym := string(r); sym != ImplSymbol {
					score[sym]++
				}
			}
		}
	}
	syms := make([]string, 0, len(score))
	for sym := range score {
		syms = append(syms, sym)
	}
	slices.SortFunc(syms, func(a, b string) int {
		return cmp.Or(score[b]-score[a], strings.Compare(a, b))
	})
	syms = syms[:min(n, len(syms))]
	slices.Sort(syms)
	return syms
}
