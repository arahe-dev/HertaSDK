package mutate

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Report is a finished run as saved in run.json: everything needed to
// re-triage, re-render or compare it without running a mutant again.
type Report struct {
	Title   string
	Stages  []Stage
	Results []Result
	Stale   []string          // triage entries that matched no survivor
	Sources map[string]string // file → sha256 of the source the verdicts are about
}

// Fingerprints hashes each source, for Report.Sources.
func Fingerprints(srcs map[string][]byte) map[string]string {
	fp := make(map[string]string, len(srcs))
	for f, b := range srcs {
		fp[f] = fmt.Sprintf("%x", sha256.Sum256(b))
	}
	return fp
}

// Retriage discards earlier equivalence verdicts and applies t.
func (r *Report) Retriage(t Triage) {
	for i := range r.Results {
		if r.Results[i].Status == Equivalent {
			r.Results[i].Status, r.Results[i].Reason = Survived, ""
		}
	}
	r.Stale = t.Apply(r.Results)
}

// Write saves run.json, mutants.csv and summary.md into dir.
func (r *Report) Write(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	j, err := json.MarshalIndent(r, "", "\t")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "run.json"), j, 0o644); err != nil {
		return err
	}
	if err := create(filepath.Join(dir, "mutants.csv"), func(f *os.File) error { return WriteCSV(f, r.Results) }); err != nil {
		return err
	}
	return create(filepath.Join(dir, "summary.md"), func(f *os.File) error {
		WriteSummary(f, r.Title, r.Stages, r.Results, r.Stale)
		return nil
	})
}

// ReadReport loads dir/run.json.
func ReadReport(dir string) (*Report, error) {
	b, err := os.ReadFile(filepath.Join(dir, "run.json"))
	if err != nil {
		return nil, err
	}
	r := new(Report)
	return r, json.Unmarshal(b, r)
}

func create(name string, fill func(*os.File) error) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	if err := fill(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Resume returns the mutants of ms that r has not settled: those it never
// judged, its survivors (equivalent or not), and every mutant of a file
// whose source differs from srcs. Kills and compile errors of an unchanged
// file stand, which holds as long as stages are only ever added.
func (r *Report) Resume(ms []Mutant, srcs map[string][]byte) []Mutant {
	old, now := r.byKey(), Fingerprints(srcs)
	var todo []Mutant
	for _, m := range ms {
		o, ok := old[m.Key()]
		if !ok || r.Sources[m.File] != now[m.File] || o.Status != Killed && o.Status != Compile {
			todo = append(todo, m)
		}
	}
	return todo
}

// Merge returns one result per mutant of ms, in order: the fresh verdict
// where it was judged again, else r's earlier one, renumbered to ms.
func (r *Report) Merge(ms []Mutant, fresh []Result) []Result {
	old, now := r.byKey(), (&Report{Results: fresh}).byKey()
	out := make([]Result, 0, len(ms))
	for _, m := range ms {
		res, ok := now[m.Key()]
		if !ok {
			res = old[m.Key()]
		}
		res.Mutant = m
		out = append(out, res)
	}
	return out
}

func (r *Report) byKey() map[string]Result {
	m := make(map[string]Result, len(r.Results))
	for _, res := range r.Results {
		m[res.Key()] = res
	}
	return m
}
