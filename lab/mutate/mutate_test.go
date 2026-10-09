package mutate

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const fixture = `package p

func (r *R) acquire(n int) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cur+n <= r.size && !r.closed {
		r.cur += n
		return true, nil
	}
	return false, errFull
}

func pair(a, b func()) {
	defer a()
	defer b()
}
`

// TestGenerateOperators pins the exact mutants each operator yields on a
// fixture that exercises all of them.
func TestGenerateOperators(t *testing.T) {
	ms, err := Generate("p/r.go", []byte(fixture))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range ms {
		got = append(got, string(m.Op)+": "+m.From+" → "+m.To)
	}
	want := []string{
		"stmt-delete: r.mu.Lock() → ",
		"defer-delete: defer r.mu.Unlock() → ",
		"defer-inline: defer → ",
		"cond-negate: <= → >",
		"cond-boundary: <= → <",
		"cond-logic: && → ||",
		"cond-not: ! → ",
		"stmt-delete: r.cur += n → ",
		"bool-flip: true → false",
		"bool-flip: false → true",
		"err-nil: errFull → nil",
		"defer-delete: defer a() → ",
		"defer-inline: defer → ",
		"defer-delete: defer b() → ",
		"defer-inline: defer → ",
		"stmt-swap: swap → defer a() ⇄ defer b()",
	}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("missing mutant %q", w)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d mutants, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
}

// TestMutantsParseAndKeepLines checks every splice yields valid Go whose
// line count is unchanged unless statements were swapped across lines.
func TestMutantsParseAndKeepLines(t *testing.T) {
	ms, _ := Generate("p/r.go", []byte(fixture))
	for _, m := range ms {
		out := m.Apply([]byte(fixture))
		if _, err := parser.ParseFile(token.NewFileSet(), "", out, 0); err != nil {
			t.Errorf("%s %s: mutant does not parse: %v", m.Where(), m.Op, err)
		}
		if bytes.Count(out, []byte("\n")) != strings.Count(fixture, "\n") {
			t.Errorf("%s %s: line count changed", m.Where(), m.Op)
		}
	}
}

func TestIntOffAndDedupe(t *testing.T) {
	src := "package p\n\nconst a, b = 0x10, 1_000\n\nfunc f() {\n\tg()\n\tg()\n}\n"
	ms, err := Generate("p/c.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	var tos []string
	keys := map[string]bool{}
	for _, m := range ms {
		if m.Op == IntOff {
			tos = append(tos, m.To)
		}
		if keys[m.Key()] {
			t.Errorf("duplicate key %s", m.Key())
		}
		keys[m.Key()] = true
	}
	if !slices.Equal(tos, []string{"17", "15", "1001", "999"}) {
		t.Errorf("int-off = %v", tos)
	}
	// Swapping g() with g() reproduces the source, and deleting either g()
	// yields the same token stream: one delete, no swap.
	if n := count(ms, StmtDelete); n != 1 {
		t.Errorf("stmt-delete mutants = %d, want 1 after dedupe", n)
	}
	if n := count(ms, StmtSwap); n != 0 {
		t.Errorf("stmt-swap mutants = %d, want 0", n)
	}
}

func count(ms []Mutant, op Operator) (n int) {
	for _, m := range ms {
		if m.Op == op {
			n++
		}
	}
	return n
}

func TestDiff(t *testing.T) {
	a := "a\nb\nc\nd\n"
	b := "a\nb\nX\nY\nd\n"
	want := "--- f\n+++ f (mutant)\n@@ -2,3 +2,4 @@\n b\n-c\n+X\n+Y\n d\n"
	if got := Diff("f", []byte(a), []byte(b)); got != want {
		t.Errorf("Diff =\n%s\nwant\n%s", got, want)
	}
}

func TestParseStage(t *testing.T) {
	st, err := ParseStage("lab: LAB_IMPLS=recon X_Y=1 go test -count=1 ./lab/...")
	if err != nil {
		t.Fatal(err)
	}
	if st.Name != "lab" || !slices.Equal(st.Env, []string{"LAB_IMPLS=recon", "X_Y=1"}) ||
		!slices.Equal(st.Args, []string{"go", "test", "-count=1", "./lab/..."}) {
		t.Errorf("ParseStage = %+v", st)
	}
	for _, bad := range []string{"go test", "x:", ": go test", "x: A=1"} {
		if _, err := ParseStage(bad); err == nil {
			t.Errorf("ParseStage(%q) accepted", bad)
		}
	}
}

func TestVerdict(t *testing.T) {
	cases := []struct {
		out, how string
		killers  []string
	}{
		{"--- FAIL: TestA (0.1s)\n    --- FAIL: TestA/sub (0s)\nFAIL\n", "fail", []string{"TestA", "TestA/sub"}},
		{"panic: test timed out after 1m0s\n\trunning tests:\n\t\tTestHang (1m0s)\n", "timeout", []string{"TestHang"}},
		{"fatal error: concurrent map read and map write\n", "crash", nil},
		{"--- FAIL: TestP (0s)\npanic: boom [recovered]\n", "panic", []string{"TestP"}},
		{"FAIL\tx [build failed]\n", "build", nil},
	}
	for _, c := range cases {
		how, ks := verdict([]byte(c.out), false)
		if how != c.how || !slices.Equal(ks, c.killers) {
			t.Errorf("verdict(%q) = %s %v, want %s %v", c.out, how, ks, c.how, c.killers)
		}
	}
	if how, _ := verdict(nil, true); how != "timeout" {
		t.Errorf("hung run = %s, want timeout", how)
	}
}

func TestTriage(t *testing.T) {
	name := filepath.Join(t.TempDir(), "eq.tsv")
	os.WriteFile(name, []byte("# comment\n\nk1\tequivalent because\nk2\tstale\n"), 0o644)
	tr, err := ReadTriage(name)
	if err != nil {
		t.Fatal(err)
	}
	rs := []Result{{Status: Survived}, {Status: Survived}}
	rs[0].File, rs[1].File = "k1", "other"
	tr = Triage{rs[0].Key(): tr["k1"], "other|f|op|a|b|0": tr["k2"], "elsewhere.go|f|op|a|b|0": "not this run"}
	stale := tr.Apply(rs)
	if rs[0].Status != Equivalent || rs[0].Reason != "equivalent because" || rs[1].Status != Survived {
		t.Errorf("Apply: %+v", rs)
	}
	if !slices.Equal(stale, []string{"other|f|op|a|b|0"}) {
		t.Errorf("stale = %v", stale)
	}
	all, _ := Tallies(rs)
	if all.Score() != 0 || all.Equivalent != 1 {
		t.Errorf("tally %+v", all)
	}
}

// TestReportRoundTrip checks a saved run re-triages without re-running:
// an equivalence verdict that is withdrawn turns back into a survivor.
func TestReportRoundTrip(t *testing.T) {
	m := Mutant{ID: 3, File: "r/a.go", Line: 9, Func: "f", Op: CondNot, From: "!", To: ""}
	rep := &Report{Title: "t", Results: []Result{
		{Mutant: m, Status: Survived, Diff: "d"},
		{Mutant: Mutant{ID: 4}, Status: Killed, Stage: "own", Killers: []string{"TestA"}},
	}}
	rep.Retriage(Triage{m.Key(): "proof"})
	dir := t.TempDir()
	if err := rep.Write(dir); err != nil {
		t.Fatal(err)
	}
	back, err := ReadReport(dir)
	if err != nil {
		t.Fatal(err)
	}
	if r := back.Results[0]; r.Status != Equivalent || r.Reason != "proof" || r.Diff != "d" || r.Key() != m.Key() {
		t.Errorf("round trip lost the verdict: %+v", r)
	}
	back.Retriage(Triage{})
	if back.Results[0].Status != Survived || back.Results[1].Killers[0] != "TestA" {
		t.Errorf("retriage: %+v", back.Results)
	}
	for _, f := range []string{"mutants.csv", "summary.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Error(err)
		}
	}
}

// TestResume: a resumed run re-judges only survivors and unseen mutants
// of unchanged files, and the merge keeps every earlier kill. Once the
// file changes, nothing about it is trusted.
func TestResume(t *testing.T) {
	ms, _ := Generate("p/r.go", []byte(fixture))
	srcs := map[string][]byte{"p/r.go": []byte(fixture)}
	prev := &Report{Sources: Fingerprints(srcs), Results: []Result{
		{Mutant: ms[0], Status: Killed, Stage: "own"},
		{Mutant: ms[1], Status: Survived},
		{Mutant: ms[2], Status: Equivalent},
	}}
	todo := prev.Resume(ms, srcs)
	if len(todo) != len(ms)-1 || todo[0].Key() != ms[1].Key() {
		t.Fatalf("Resume = %d mutants starting %s, want %d starting %s", len(todo), todo[0].Key(), len(ms)-1, ms[1].Key())
	}
	edited := map[string][]byte{"p/r.go": []byte(fixture + "\n")}
	if n := len(prev.Resume(ms, edited)); n != len(ms) {
		t.Errorf("after an edit Resume = %d mutants, want all %d", n, len(ms))
	}
	fresh := []Result{{Mutant: ms[1], Status: Killed, Stage: "race"}}
	got := prev.Merge(ms, fresh)
	if len(got) != len(ms) || got[0].Stage != "own" || got[1].Stage != "race" || got[3].Status != "" {
		t.Errorf("Merge = %+v", got[:4])
	}
}
