// Command mutate mutation-tests one Go package against a cascade of test
// stages, re-triages saved runs, and compares them.
//
//	mutate run -root . -pkg recon -out OUT -triage equivalent.tsv \
//	    -stage 'own: go test -count=1 -short ./recon/' \
//	    -stage 'lab: LAB_IMPLS=recon go test -count=1 ./lab/assume'
//	mutate report -out OUT -triage equivalent.tsv
//	mutate compare orig=OUT1 recon=OUT2
//
// run and report write OUT/run.json, OUT/mutants.csv and OUT/summary.md,
// and exit 1 while a mutant survives that the triage file does not prove
// equivalent, so they can gate CI.
package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/arahe-dev/hertasdk/lab/mutate"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: mutate run|report|compare ...")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "run":
		err = run(os.Args[2:])
	case "report":
		err = report(os.Args[2:])
	case "compare":
		err = compare(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "mutate:", err)
		os.Exit(1)
	}
}

type stages []mutate.Stage

func (s *stages) String() string { return fmt.Sprint(*s) }
func (s *stages) Set(v string) error {
	st, err := mutate.ParseStage(v)
	*s = append(*s, st)
	return err
}

func run(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var cfg mutate.Config
	var sts stages
	fs.StringVar(&cfg.Root, "root", ".", "module root to copy into each workspace")
	fs.StringVar(&cfg.Pkg, "pkg", ".", "package directory to mutate, relative to -root")
	fs.Var(&sts, "stage", "kill-cascade stage \"name: [ENV=V ...] cmd args\" (repeatable, in order)")
	fs.StringVar(&cfg.Work, "work", filepath.Join(os.TempDir(), "hertamut"), "scratch directory for workspaces")
	fs.IntVar(&cfg.Jobs, "j", 6, "parallel workspaces")
	fs.DurationVar(&cfg.Hang, "hang", 4*time.Minute, "per-command wall-clock limit; a mutant past it is killed as hung")
	out := fs.String("out", "mutants-out", "directory for run.json, mutants.csv and summary.md")
	title := fs.String("title", "", "report title (default: -pkg)")
	triage := fs.String("triage", "", "triage file of proven-equivalent mutants (key<TAB>reason)")
	ops := fs.String("ops", "", "comma list of operators to run (default: all)")
	files := fs.String("files", "", "regexp: only mutate matching file paths")
	ids := fs.String("ids", "", "comma list of mutant ids to run (default: all)")
	list := fs.Bool("list", false, "list the mutants and exit")
	resume := fs.Bool("resume", false, "keep -out's earlier kills and compile errors; judge only its survivors and new mutants")
	fs.Parse(args)
	cfg.Log = os.Stderr

	ms, srcs, err := mutate.Load(cfg.Root, cfg.Pkg)
	if err != nil {
		return err
	}
	if ms, err = selectMutants(ms, *ops, *files, *ids); err != nil {
		return err
	}
	if *list {
		for _, m := range ms {
			fmt.Printf("#%d\t%s\t%s\t%s\t%s → %s\n", m.ID, m.Where(), m.Func, m.Op, m.From, m.To)
		}
		return nil
	}
	if len(sts) == 0 {
		return fmt.Errorf("no -stage given")
	}
	cfg.Stages = sts

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	start := time.Now()
	prev := &mutate.Report{}
	if *resume {
		if prev, err = mutate.ReadReport(*out); err != nil {
			return err
		}
	}
	var rs []mutate.Result
	todo := prev.Resume(ms, srcs)
	if len(todo) > 0 {
		if rs, err = mutate.Run(ctx, cfg, todo, srcs); err != nil {
			return err
		}
	}
	rs = prev.Merge(ms, rs)
	fmt.Fprintf(os.Stderr, "%d mutants (%d judged) in %s\n", len(rs), len(todo), time.Since(start).Round(time.Second))
	rep := &mutate.Report{Title: cmp.Or(*title, cfg.Pkg), Stages: sts, Results: rs, Sources: mutate.Fingerprints(srcs)}
	return finish(rep, *out, *triage)
}

// report re-applies a triage file to a saved run and re-renders it.
func report(args []string) error {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	out := fs.String("out", "mutants-out", "directory holding run.json")
	triage := fs.String("triage", "", "triage file of proven-equivalent mutants (key<TAB>reason)")
	fs.Parse(args)
	rep, err := mutate.ReadReport(*out)
	if err != nil {
		return err
	}
	return finish(rep, *out, *triage)
}

func finish(rep *mutate.Report, out, triage string) error {
	t, err := mutate.ReadTriage(triage)
	if err != nil {
		return err
	}
	rep.Retriage(t)
	if err := rep.Write(out); err != nil {
		return err
	}
	all, _ := mutate.Tallies(rep.Results)
	fmt.Fprintf(os.Stderr, "%s: %d killed, %d survived, %d equivalent, %d compile; score %.1f%%\n",
		rep.Title, all.Killed, all.Survived, all.Equivalent, all.Compile, 100*all.Score())
	if all.Survived > 0 {
		return fmt.Errorf("%d mutants survived; see %s", all.Survived, filepath.Join(out, "summary.md"))
	}
	return nil
}

func selectMutants(ms []mutate.Mutant, ops, files, ids string) ([]mutate.Mutant, error) {
	re, err := regexp.Compile(files)
	if err != nil {
		return nil, err
	}
	keepOp := strings.Split(ops, ",")
	keepID := strings.Split(ids, ",")
	ms = slices.DeleteFunc(ms, func(m mutate.Mutant) bool {
		return !re.MatchString(m.File) ||
			ops != "" && !slices.Contains(keepOp, string(m.Op)) ||
			ids != "" && !slices.Contains(keepID, fmt.Sprint(m.ID))
	})
	if len(ms) == 0 {
		return nil, fmt.Errorf("no mutants selected")
	}
	return ms, nil
}

// compare prints a per-operator score table for NAME=DIR arguments.
func compare(args []string) error {
	var names []string
	var runs [][]mutate.Result
	for _, a := range args {
		name, dir, ok := strings.Cut(a, "=")
		if !ok {
			return fmt.Errorf("compare: want NAME=DIR, got %q", a)
		}
		rep, err := mutate.ReadReport(dir)
		if err != nil {
			return err
		}
		names, runs = append(names, name), append(runs, rep.Results)
	}
	mutate.WriteCompare(os.Stdout, names, runs)
	return nil
}
