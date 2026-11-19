// Package hertatest runs Herta golden-file test cases under hard caps on
// suite size and runtime. It exists so the conformance suite stays small and
// agent friendly: one command, plain-text goldens, and hard limits that fail
// the run before iteration can blow up CI.
//
// Layout of a case directory (lang/testdata/cases/<name>/):
//
//	input.herta    source program under test (required)
//	expected.txt   expected stdout of the check command (required)
//	expected.exit  expected exit code when non-zero (optional)
//	args.txt       extra arguments appended to the command (optional)
//	skip           file whose presence skips the case (optional)
package hertatest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Limits that keep the suite small. All are enforced; exceeding any of them
// fails the run, not just the case.
type Limits struct {
	MaxCases      int           // hard cap on discovered cases
	CaseTimeout   time.Duration // per-case wall clock
	TotalTimeout  time.Duration // whole-suite wall clock
	MaxFileBytes  int64         // largest allowed file in a case dir
	MaxOutputSize int           // captured stdout+stderr cap per case
}

// DefaultLimits are the caps CI runs with. Bumping them is a deliberate
// decision, not something agent iterations should do silently.
var DefaultLimits = Limits{
	MaxCases:      200,
	CaseTimeout:   5 * time.Second,
	TotalTimeout:  60 * time.Second,
	MaxFileBytes:  64 << 10,
	MaxOutputSize: 1 << 20,
}

// Selftest is the built-in exec mode: it echoes input.herta to stdout, so the
// harness itself is exercised before a real checker exists. CI uses it until
// lang/cmd/herta lands.
const Selftest = "selftest"

// Case is one discovered test case.
type Case struct {
	Name    string
	Dir     string
	Input   string // path to input.herta
	Golden  string // path to expected.txt
	Exit    string // path to expected.exit, "" if absent
	Args    []string
	Skipped bool
}

// Result is the outcome of running one case.
type Result struct {
	Case    Case
	Skipped bool
	Elapsed time.Duration
	Err     error // nil on pass; describes mismatch or run failure
}

// Discover finds case directories under dir, enforcing the file-size and
// case-count caps. Results are sorted by name for deterministic output.
func Discover(dir string, lim Limits) ([]Case, error) {
	var cases []Case
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > lim.MaxFileBytes {
			return fmt.Errorf("%s exceeds the %d-byte case file cap", path, lim.MaxFileBytes)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading cases dir: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			return nil, fmt.Errorf("%s: cases dir holds directories only, found file %q", dir, e.Name())
		}
		caseDir := filepath.Join(dir, e.Name())
		c := Case{
			Name:   e.Name(),
			Dir:    caseDir,
			Input:  filepath.Join(caseDir, "input.herta"),
			Golden: filepath.Join(caseDir, "expected.txt"),
		}
		for _, f := range []string{c.Input, c.Golden} {
			if _, err := os.Stat(f); err != nil {
				return nil, fmt.Errorf("case %s: %v", c.Name, err)
			}
		}
		if _, err := os.Stat(filepath.Join(caseDir, "expected.exit")); err == nil {
			c.Exit = filepath.Join(caseDir, "expected.exit")
		}
		if _, err := os.Stat(filepath.Join(caseDir, "skip")); err == nil {
			c.Skipped = true
		}
		if b, err := os.ReadFile(filepath.Join(caseDir, "args.txt")); err == nil {
			c.Args = strings.Fields(string(b))
		}
		cases = append(cases, c)
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
	if len(cases) > lim.MaxCases {
		return nil, fmt.Errorf("%d cases exceed the suite cap of %d; split the suite or raise the cap deliberately", len(cases), lim.MaxCases)
	}
	return cases, nil
}

// Run executes every case with exec as the check command and returns the
// per-case results. exec is a shell-free command template: "{file}" expands
// to the case's input path and "{args}" to its args.txt contents; the literal
// word Selftest selects the built-in echo engine.
func Run(ctx context.Context, cases []Case, execTemplate string, lim Limits) []Result {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(lim.TotalTimeout))
	defer cancel()
	results := make([]Result, 0, len(cases))
	for _, c := range cases {
		if c.Skipped {
			results = append(results, Result{Case: c, Skipped: true})
			continue
		}
		start := time.Now()
		err := runCase(ctx, c, execTemplate, lim)
		results = append(results, Result{Case: c, Elapsed: time.Since(start), Err: err})
		if ctx.Err() != nil {
			break
		}
	}
	return results
}

func runCase(ctx context.Context, c Case, execTemplate string, lim Limits) error {
	want, err := os.ReadFile(c.Golden)
	if err != nil {
		return err
	}
	var got []byte
	var runErr error
	if execTemplate == Selftest {
		got, runErr = os.ReadFile(c.Input)
	} else {
		got, runErr = execCase(ctx, c, execTemplate, lim)
	}
	if err := checkExit(c, runErr); err != nil {
		return err
	}
	if runErr != nil && c.Exit == "" {
		return runErr
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("golden mismatch\n--- expected (%s) ---\n%s\n--- got ---\n%s", c.Golden, want, got)
	}
	return nil
}

func execCase(ctx context.Context, c Case, tmpl string, lim Limits) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, lim.CaseTimeout)
	defer cancel()
	fields := strings.Fields(tmpl)
	expanded := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.ReplaceAll(f, "{file}", c.Input)
		f = strings.ReplaceAll(f, "{args}", strings.Join(c.Args, " "))
		expanded = append(expanded, f)
	}
	cmd := exec.CommandContext(cctx, expanded[0], expanded[1:]...)
	var out cappedBuffer
	out.limit = lim.MaxOutputSize
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if cctx.Err() == context.DeadlineExceeded {
		return out.buf.Bytes(), fmt.Errorf("exceeded the %s per-case timeout", lim.CaseTimeout)
	}
	if out.overflowed {
		return out.buf.Bytes(), fmt.Errorf("output exceeded the %d-byte capture cap", lim.MaxOutputSize)
	}
	return out.buf.Bytes(), err
}

func checkExit(c Case, runErr error) error {
	if c.Exit != "" {
		b, err := os.ReadFile(c.Exit)
		if err != nil {
			return err
		}
		wantCode := strings.TrimSpace(string(b))
		var ee *exec.ExitError
		if errors.As(runErr, &ee) && fmt.Sprint(ee.ExitCode()) == wantCode {
			return nil
		}
		if runErr == nil {
			return fmt.Errorf("expected exit code %s, command succeeded", wantCode)
		}
		return fmt.Errorf("expected exit code %s: %w", wantCode, runErr)
	}
	return nil
}

type cappedBuffer struct {
	buf        bytes.Buffer
	limit      int
	overflowed bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > b.limit {
		b.overflowed = true
		b.buf.Write(p[:max(0, b.limit-b.buf.Len())])
		return len(p), nil
	}
	return b.buf.Write(p)
}

// Summary formats results for CI output and reports whether the run passed.
func Summary(results []Result) (string, bool) {
	var b strings.Builder
	var pass, fail, skip int
	for _, r := range results {
		switch {
		case r.Skipped:
			skip++
			fmt.Fprintf(&b, "SKIP %s\n", r.Case.Name)
		case r.Err == nil:
			pass++
			fmt.Fprintf(&b, "ok   %s (%s)\n", r.Case.Name, r.Elapsed.Round(time.Millisecond))
		default:
			fail++
			fmt.Fprintf(&b, "FAIL %s: %v\n", r.Case.Name, r.Err)
		}
	}
	fmt.Fprintf(&b, "%d passed, %d failed, %d skipped\n", pass, fail, skip)
	return b.String(), fail == 0
}
