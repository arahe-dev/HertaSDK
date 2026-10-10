package main

import (
	"strings"
	"testing"
)

const base = `{"Action":"output","Package":"m/a","Test":"TestDB","Output":"    db_test.go:9: DATABASE_URL not set\n"}
{"Action":"skip","Package":"m/a","Test":"TestDB"}
{"Action":"pass","Package":"m/a","Test":"TestX"}
{"Action":"pass","Package":"m/a","Test":"TestX"}
{"Action":"pass","Package":"m/a","Test":"TestY"}
{"Action":"pass","Package":"m/a","Elapsed":1.5}
{"Action":"skip","Package":"m/cmd"}
`

// TestRenderFlagsRegressions: a test that passes in the baseline but
// fails in the candidate is a regression; one that fails only in the
// baseline is an improvement, reported but not counted.
func TestRenderFlagsRegressions(t *testing.T) {
	cand := strings.NewReplacer(
		`"pass","Package":"m/a","Test":"TestY"`, `"fail","Package":"m/a","Test":"TestY"`,
		`"pass","Package":"m/a","Elapsed"`, `"fail","Package":"m/a","Elapsed"`,
	).Replace(base)
	a, err := parse(strings.NewReader(base))
	if err != nil {
		t.Fatal(err)
	}
	b, err := parse(strings.NewReader(cand))
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	if n := render(&sb, "v0.2", "recon", "m/", a, b); n != 2 {
		t.Errorf("regressions = %d, want 2 (package and TestY)\n%s", n, sb.String())
	}
	out := sb.String()
	for _, want := range []string{
		"| a | ok | **FAIL** | 2/0/1 | 1/1/1 | 1.5 | 1.5 |",
		"| a | TestY | pass | FAIL |",
		"1 packages have no test files.",
		"- 1 tests: `db_test.go:9: DATABASE_URL not set`\n  - a.TestDB",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	if strings.Contains(out, "TestX |") {
		t.Errorf("TestX passed twice in both runs; it must not be listed as a difference")
	}

	sb.Reset()
	if n := render(&sb, "v0.2", "recon", "m/", b, a); n != 0 {
		t.Errorf("improvements counted as regressions: %d", n)
	}
}

func TestTallyString(t *testing.T) {
	for _, c := range []struct {
		t    tally
		want string
	}{{tally{}, "absent"}, {tally{pass: 1}, "pass"}, {tally{pass: 2, fail: 1}, "FAIL 2/3"}, {tally{skip: 3}, "skip"}} {
		if got := c.t.String(); got != c.want {
			t.Errorf("%+v = %q, want %q", c.t, got, c.want)
		}
	}
}
