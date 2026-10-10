package doe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestWriteProducesAllArtifacts(t *testing.T) {
	x, trials := synthetic(3, 0.5, 4)
	for i := range x.Arms {
		x.Arms[i].Impl = nopImpl{}
	}
	dir := t.TempDir()
	if err := Write(dir, x, trials, Analyze(x, trials), Meta{ID: "t", Kind: "screen", Reps: 3}); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"runs.csv", "effects.csv", "summary.json", "REPORT.md"} {
		if fi, err := os.Stat(filepath.Join(dir, f)); err != nil || fi.Size() == 0 {
			t.Errorf("%s missing or empty: %v", f, err)
		}
	}
	runs, _ := os.ReadFile(filepath.Join(dir, "runs.csv"))
	if n := strings.Count(string(runs), "\n"); n != len(trials)+1 {
		t.Errorf("runs.csv has %d lines, want %d", n, len(trials)+1)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "summary.json"))
	var s Summary
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("summary.json: %v", err)
	}
	if s.Design == nil || s.Design.Resolution != 4 || len(s.Design.Generators) != 5 {
		t.Errorf("design %+v", s.Design)
	}
	report, _ := os.ReadFile(filepath.Join(dir, "REPORT.md"))
	for _, want := range []string{"## Verdict", "## Noise", "## Design", "## Effects", "F=ABCD"} {
		if !strings.Contains(string(report), want) {
			t.Errorf("REPORT.md lacks %q", want)
		}
	}
}

func TestActiveFactorsRanksSignificantSymbols(t *testing.T) {
	s := Summary{TopEffects: map[string][]effectJSON{
		"lat_p99":    {{Effect: "A", Significant: true}, {Effect: "AC=BD", Significant: true}, {Effect: "CX", Significant: true}},
		"throughput": {{Effect: "C", Significant: true}, {Effect: "E", Significant: false}},
		"lat_max":    {{Effect: "K", Significant: true}}, // not a key response
	}}
	if got := ActiveFactors(s, 2); !slices.Equal(got, []string{"A", "C"}) {
		t.Errorf("got %v, want [A C]", got)
	}
}
