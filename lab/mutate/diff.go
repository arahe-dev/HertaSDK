package mutate

import (
	"fmt"
	"strings"
)

// Diff renders a mutant as one unified hunk with a line of context. A
// mutant is one contiguous change, so trimming the common prefix and
// suffix lines yields the minimal hunk without a general diff algorithm.
func Diff(name string, a, b []byte) string {
	x, y := lines(a), lines(b)
	pre := 0
	for pre < len(x) && pre < len(y) && x[pre] == y[pre] {
		pre++
	}
	suf := 0
	for suf < len(x)-pre && suf < len(y)-pre && x[len(x)-1-suf] == y[len(y)-1-suf] {
		suf++
	}
	lo, xEnd, yEnd := max(pre-1, 0), len(x)-suf, len(y)-suf
	hi := min(xEnd+1, len(x)) // one line of trailing context

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s (mutant)\n@@ -%d,%d +%d,%d @@\n",
		name, name, lo+1, hi-lo, lo+1, hi-lo+(yEnd-xEnd))
	put := func(mark string, ls []string) {
		for _, l := range ls {
			sb.WriteString(mark + l + "\n")
		}
	}
	put(" ", x[lo:pre])
	put("-", x[pre:xEnd])
	put("+", y[pre:yEnd])
	put(" ", x[xEnd:hi])
	return sb.String()
}

func lines(b []byte) []string {
	return strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
}
