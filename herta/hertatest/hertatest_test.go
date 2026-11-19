package hertatest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeCase(t *testing.T, dir, name, input, golden string) {
	t.Helper()
	d := filepath.Join(dir, name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "input.he"), []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "expected.golden"), []byte(golden), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSelftestGoldenPass(t *testing.T) {
	dir := t.TempDir()
	writeCase(t, dir, "a", "op a() {}\n", "op a() {}\n")
	cases, err := Discover(dir, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	results := Run(context.Background(), cases, Selftest, DefaultLimits)
	if results[0].Err != nil {
		t.Fatalf("expected pass: %v", results[0].Err)
	}
}

func TestGoldenMismatchFails(t *testing.T) {
	dir := t.TempDir()
	writeCase(t, dir, "a", "op a() {}\n", "different\n")
	cases, _ := Discover(dir, DefaultLimits)
	results := Run(context.Background(), cases, Selftest, DefaultLimits)
	if results[0].Err == nil {
		t.Fatal("expected mismatch to fail")
	}
	if _, ok := Summary(results); ok {
		t.Fatal("summary should report failure")
	}
}

func TestCaseCountCap(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		writeCase(t, dir, string(rune('a'+i)), "x\n", "x\n")
	}
	lim := DefaultLimits
	lim.MaxCases = 2
	if _, err := Discover(dir, lim); err == nil {
		t.Fatal("expected case-count cap to reject the suite")
	}
}

func TestFileSizeCap(t *testing.T) {
	dir := t.TempDir()
	writeCase(t, dir, "a", "x\n", "x\n")
	lim := DefaultLimits
	lim.MaxFileBytes = 1
	if _, err := Discover(dir, lim); err == nil {
		t.Fatal("expected file-size cap to reject the suite")
	}
}

func TestPerCaseTimeout(t *testing.T) {
	dir := t.TempDir()
	writeCase(t, dir, "slow", "x\n", "x\n")
	cases, _ := Discover(dir, DefaultLimits)
	lim := DefaultLimits
	lim.CaseTimeout = 50 * time.Millisecond
	results := Run(context.Background(), cases, "sleep 2", lim)
	if results[0].Err == nil {
		t.Fatal("expected timeout failure")
	}
}

func TestSuiteTimeoutFailsRemainingCases(t *testing.T) {
	dir := t.TempDir()
	writeCase(t, dir, "slow", "x\n", "x\n")
	writeCase(t, dir, "unreached", "x\n", "x\n")
	cases, _ := Discover(dir, DefaultLimits)
	lim := DefaultLimits
	lim.TotalTimeout = 100 * time.Millisecond
	results := Run(context.Background(), cases, "sleep 2", lim)
	if len(results) != 2 {
		t.Fatalf("expected a result per case, got %d", len(results))
	}
	for _, r := range results {
		if r.Err == nil {
			t.Fatalf("case %s should have failed", r.Case.Name)
		}
	}
	if _, ok := Summary(results); ok {
		t.Fatal("an expired suite must not report a clean pass")
	}
}

func TestArgsExpandAsSeparateArguments(t *testing.T) {
	dir := t.TempDir()
	writeCase(t, dir, "a", "x\n", "<one><two>")
	if err := os.WriteFile(filepath.Join(dir, "a", "args.txt"), []byte("one two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases, err := Discover(dir, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	// printf reuses its format per argument: separate args print <one><two>,
	// a merged "one two" argument would print <one two>.
	results := Run(context.Background(), cases, "printf <%s> {args}", DefaultLimits)
	if results[0].Err != nil {
		t.Fatalf("expected pass: %v", results[0].Err)
	}
}

func TestExecTemplateAndExpectedExit(t *testing.T) {
	dir := t.TempDir()
	writeCase(t, dir, "nonzero", "x\n", "")
	d := filepath.Join(dir, "nonzero")
	if err := os.WriteFile(filepath.Join(d, "expected.exit"), []byte("1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases, err := Discover(dir, DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	results := Run(context.Background(), cases, "false", DefaultLimits)
	if results[0].Err != nil {
		t.Fatalf("expected exit-code match: %v", results[0].Err)
	}
}
