package mutate

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config describes one mutation run.
type Config struct {
	Root   string        // module tree copied into each workspace
	Pkg    string        // package directory to mutate, slash path relative to Root
	Stages []Stage       // the kill cascade, run in order
	Work   string        // scratch directory that holds the workspaces
	Jobs   int           // workspaces run in parallel
	Hang   time.Duration // wall-clock limit per command; past it the mutant hangs, i.e. is killed
	Log    io.Writer     // one progress line per mutant
}

// Status is a mutant's verdict.
type Status string

const (
	Killed     Status = "killed"     // a stage failed or hung
	Survived   Status = "survived"   // every stage passed
	Equivalent Status = "equivalent" // survived, and the triage file proves no test can kill it
	Compile    Status = "compile"    // the package no longer builds
)

// Result is a mutant and its verdict.
type Result struct {
	Mutant
	Status  Status
	Stage   string   // the stage that killed it
	How     string   // fail, timeout, panic, crash or build
	Killers []string // the failing or hung tests
	Flaky   []string // "stage: test" failures that did not reproduce, so killed nothing
	Elapsed time.Duration
	Diff    string
	Reason  string // why an Equivalent mutant cannot be killed
}

// Load reads the package's non-test files and enumerates their mutants,
// numbered from 1. It also returns the sources, keyed by slash path.
func Load(root, pkg string) ([]Mutant, map[string][]byte, error) {
	names, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pkg), "*.go"))
	if err != nil {
		return nil, nil, err
	}
	srcs := map[string][]byte{}
	var all []Mutant
	for _, n := range names {
		if strings.HasSuffix(n, "_test.go") {
			continue
		}
		rel := path.Join(pkg, filepath.Base(n))
		src, err := os.ReadFile(n)
		if err != nil {
			return nil, nil, err
		}
		ms, err := Generate(rel, src)
		if err != nil {
			return nil, nil, err
		}
		srcs[rel] = src
		all = append(all, ms...)
	}
	for i := range all {
		all[i].ID = i + 1
	}
	return all, srcs, nil
}

// Run judges every mutant. It first proves the unmutated tree passes
// every stage, so a kill always means the mutant changed behavior.
func Run(ctx context.Context, cfg Config, ms []Mutant, srcs map[string][]byte) ([]Result, error) {
	work, err := filepath.Abs(cfg.Work)
	if err != nil {
		return nil, err
	}
	spaces := make([]string, max(cfg.Jobs, 1))
	for i := range spaces {
		spaces[i] = filepath.Join(work, fmt.Sprintf("w%02d", i))
		if err := os.RemoveAll(spaces[i]); err != nil {
			return nil, err
		}
		if err := copyTree(cfg.Root, spaces[i], work); err != nil {
			return nil, err
		}
	}
	if err := baseline(ctx, cfg, spaces[0]); err != nil {
		return nil, err
	}

	results := make([]Result, len(ms))
	next := make(chan int)
	var mu sync.Mutex
	done := 0
	var wg sync.WaitGroup
	for _, ws := range spaces {
		wg.Go(func() {
			for i := range next {
				r := judge(ctx, cfg, ws, ms[i], srcs[ms[i].File])
				results[i] = r
				mu.Lock()
				done++
				fmt.Fprintf(cfg.Log, "[%d/%d] #%d %s %s %s\n", done, len(ms), r.ID, r.Where(), r.Op, describe(r))
				mu.Unlock()
			}
		})
	}
	for i := range ms {
		select {
		case next <- i:
		case <-ctx.Done():
		}
	}
	close(next)
	wg.Wait()
	return results, ctx.Err()
}

func describe(r Result) string {
	if r.Status != Killed {
		return fmt.Sprintf("%s %.1fs", r.Status, r.Elapsed.Seconds())
	}
	return fmt.Sprintf("killed(%s/%s %s) %.1fs", r.Stage, r.How, strings.Join(r.Killers, ","), r.Elapsed.Seconds())
}

func baseline(ctx context.Context, cfg Config, ws string) error {
	cmds := append([]Stage{buildStage(cfg.Pkg)}, cfg.Stages...)
	for _, st := range cmds {
		start := time.Now()
		out, hung, err := execute(ctx, ws, st.Env, st.Args, cfg.Hang)
		if err != nil {
			return fmt.Errorf("baseline %q fails on the unmutated tree (hung=%v): %v\n%s", st.Name, hung, err, tail(out, 40))
		}
		fmt.Fprintf(cfg.Log, "baseline %s: ok %.1fs\n", st.Name, time.Since(start).Seconds())
	}
	return nil
}

func buildStage(pkg string) Stage {
	return Stage{Name: "build", Args: []string{"go", "build", "./" + strings.TrimPrefix(pkg, "./")}}
}

// judge applies m in workspace ws, runs the cascade, and restores the file.
func judge(ctx context.Context, cfg Config, ws string, m Mutant, src []byte) (r Result) {
	r.Mutant = m
	mut := m.Apply(src)
	r.Diff = Diff(m.File, src, mut)
	file := filepath.Join(ws, filepath.FromSlash(m.File))
	start := time.Now()
	defer func() { r.Elapsed = time.Since(start) }()
	if err := os.WriteFile(file, mut, 0o644); err != nil {
		panic(err) // the workspace is ours; failing to write it is a broken run
	}
	defer func() {
		if err := os.WriteFile(file, src, 0o644); err != nil {
			panic(err)
		}
	}()

	b := buildStage(cfg.Pkg)
	if _, _, err := execute(ctx, ws, nil, b.Args, cfg.Hang); err != nil {
		r.Status = Compile
		return
	}
	for _, st := range cfg.Stages {
		out, hung, err := execute(ctx, ws, st.Env, st.Args, cfg.Hang)
		if err == nil {
			continue
		}
		how, killers := verdict(out, hung)
		if how == "fail" && !fails(ctx, cfg, ws, st) {
			for _, k := range killers {
				r.Flaky = append(r.Flaky, st.Name+": "+k)
			}
			continue
		}
		r.Status, r.Stage, r.How, r.Killers = Killed, st.Name, how, killers
		return
	}
	r.Status = Survived
	return
}

// fails re-runs a whole stage. A plain test failure kills a mutant only
// if the stage fails again, so a timing assertion tripped by the load of
// parallel workers kills nothing. The stage is re-run as is, not narrowed
// to the failing tests: a test that depends on what ran before it would
// fail alone and confirm a kill the mutant did not cause. Hangs, panics
// and crashes are not re-run; within the stage's timeout they are never
// load artifacts of passing code.
func fails(ctx context.Context, cfg Config, ws string, st Stage) bool {
	_, _, err := execute(ctx, ws, st.Env, st.Args, cfg.Hang)
	return err != nil
}

// copyTree copies the module at src into dst, skipping dot directories
// (.git) and skip itself when the scratch area lives inside src.
func copyTree(src, dst, skip string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			abs, _ := filepath.Abs(p)
			if p != src && (strings.HasPrefix(d.Name(), ".") || abs == skip) {
				return filepath.SkipDir
			}
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		to := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(to, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(to, b, 0o644)
	})
}

func tail(out []byte, n int) string {
	ls := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	return strings.Join(ls[max(len(ls)-n, 0):], "\n")
}
