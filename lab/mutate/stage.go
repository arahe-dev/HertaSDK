package mutate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Stage is one test command in the kill cascade.
type Stage struct {
	Name string
	Env  []string // KEY=VALUE, layered over the inherited environment
	Args []string // argv, run from the workspace root
}

var envAssign = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// ParseStage reads "name: [KEY=VALUE ...] command args...". Arguments are
// split on whitespace; there is no quoting, so no shell is involved.
func ParseStage(s string) (Stage, error) {
	name, cmd, ok := strings.Cut(s, ":")
	f := strings.Fields(cmd)
	if !ok || strings.TrimSpace(name) == "" {
		return Stage{}, fmt.Errorf("stage %q: want \"name: command args...\"", s)
	}
	st := Stage{Name: strings.TrimSpace(name)}
	for len(f) > 0 && envAssign.MatchString(f[0]) {
		st.Env, f = append(st.Env, f[0]), f[1:]
	}
	if len(f) == 0 {
		return Stage{}, fmt.Errorf("stage %q: no command", s)
	}
	st.Args = f
	return st, nil
}

func (s Stage) String() string {
	return s.Name + ": " + strings.Join(slices.Concat(s.Env, s.Args), " ")
}

// execute runs argv in dir. A run still going after hang is killed with
// its whole process tree (go test's child binary included) and reported
// as hung.
func execute(ctx context.Context, dir string, env, argv []string, hang time.Duration) (out []byte, hung bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, hang)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	ownGroup(cmd)
	cmd.Cancel = func() error { return killTree(cmd) }
	cmd.WaitDelay = 10 * time.Second
	out, err = cmd.CombinedOutput()
	return out, err != nil && ctx.Err() == context.DeadlineExceeded, err
}

var (
	failLine    = regexp.MustCompile(`(?m)^\s*--- FAIL: (\S+)`)
	runningLine = regexp.MustCompile(`(?m)^\s+(Test\S*) \([0-9.hmsµ]+\)\s*$`)
)

// verdict explains a failed stage: how the mutant was killed and by which
// tests.
func verdict(out []byte, hung bool) (how string, killers []string) {
	s := string(out)
	switch {
	case hung || strings.Contains(s, "panic: test timed out"):
		how = "timeout"
		for _, m := range runningLine.FindAllStringSubmatch(s, -1) {
			killers = append(killers, m[1])
		}
	case strings.Contains(s, "[build failed]") || strings.Contains(s, "[setup failed]"):
		how = "build"
	case strings.Contains(s, "fatal error:"):
		how = "crash"
	case strings.Contains(s, "\npanic: ") || strings.HasPrefix(s, "panic: "):
		how = "panic"
	default:
		how = "fail"
	}
	for _, m := range failLine.FindAllStringSubmatch(s, -1) {
		if !slices.Contains(killers, m[1]) {
			killers = append(killers, m[1])
		}
	}
	return how, killers
}
