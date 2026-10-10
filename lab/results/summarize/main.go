// Command summarize folds one study run (lab/results/run.sh) into a single
// summary.json: the micro-benchmark comparison, each DoE study's summary,
// the industrial and soak metrics, and how busy the machine was per stage.
//
//	go run ./results/summarize results/<run>     (from lab/)
//
// Inputs that a run did not produce are left out.
package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// studies are the DoE output directories run.sh writes, in report order.
var studies = []string{"screen", "full", "sweep", "ab", "aa-orig", "aa-recon"}

type summary struct {
	Run        string                     `json:"run"`
	Load       map[string]load            `json:"machine_cpu_percent,omitempty"`
	Bench      []map[string]string        `json:"bench,omitempty"`
	DoE        map[string]json.RawMessage `json:"doe,omitempty"`
	Industrial []json.RawMessage          `json:"industrial,omitempty"`
	Soak       []json.RawMessage          `json:"soak,omitempty"`
}

// load is whole-machine CPU use during one stage, sampled every 5s.
type load struct {
	Samples int     `json:"samples"`
	Mean    float64 `json:"mean"`
	P90     float64 `json:"p90"`
	Max     float64 `json:"max"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: summarize <run dir>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "summarize:", err)
		os.Exit(1)
	}
}

func run(dir string) error {
	s := summary{Run: filepath.Base(dir), Load: map[string]load{}, DoE: map[string]json.RawMessage{}}
	var err error
	if s.Bench, err = readCSV(filepath.Join(dir, "bench.csv")); err != nil {
		return err
	}
	for _, st := range studies {
		b, err := optional(os.ReadFile(filepath.Join(dir, st, "summary.json")))
		if err != nil {
			return err
		}
		if b != nil {
			s.DoE[st] = b
		}
	}
	if s.Industrial, err = readLines(filepath.Join(dir, "industrial.jsonl")); err != nil {
		return err
	}
	if s.Soak, err = readLines(filepath.Join(dir, "soak.jsonl")); err != nil {
		return err
	}
	cpus, _ := filepath.Glob(filepath.Join(dir, "cpu-*.csv"))
	for _, p := range cpus {
		l, err := readLoad(p)
		if err != nil {
			return err
		}
		s.Load[strings.TrimSuffix(strings.TrimPrefix(filepath.Base(p), "cpu-"), ".csv")] = l
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "summary.json"), append(b, '\n'), 0o644)
}

// optional turns a missing file into no data.
func optional(b []byte, err error) ([]byte, error) {
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return b, err
}

// readCSV returns the rows of a headed CSV as maps (nil if it is missing).
func readCSV(path string) ([]map[string]string, error) {
	b, err := optional(os.ReadFile(path))
	if b == nil {
		return nil, err
	}
	recs, err := csv.NewReader(strings.NewReader(string(b))).ReadAll()
	if err != nil || len(recs) == 0 {
		return nil, err
	}
	var rows []map[string]string
	for _, r := range recs[1:] {
		m := map[string]string{}
		for i, h := range recs[0] {
			m[h] = r[i]
		}
		rows = append(rows, m)
	}
	return rows, nil
}

// readLines returns each line of a JSON-lines file (nil if it is missing).
func readLines(path string) ([]json.RawMessage, error) {
	b, err := optional(os.ReadFile(path))
	if b == nil {
		return nil, err
	}
	var out []json.RawMessage
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			if !json.Valid([]byte(line)) {
				return nil, fmt.Errorf("%s: invalid JSON line %q", path, line)
			}
			out = append(out, json.RawMessage(line))
		}
	}
	return out, sc.Err()
}

// readLoad summarizes a typeperf CSV: a header, then "time","percent" rows
// (the first sample is blank).
func readLoad(path string) (load, error) {
	recs, err := readCSV(path)
	if err != nil {
		return load{}, err
	}
	var xs []float64
	for _, r := range recs {
		for k, v := range r {
			if strings.Contains(k, "Processor Time") {
				if x, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
					xs = append(xs, x)
				}
			}
		}
	}
	if len(xs) == 0 {
		return load{}, nil
	}
	slices.Sort(xs)
	l := load{Samples: len(xs), P90: xs[len(xs)*9/10], Max: xs[len(xs)-1]}
	for _, x := range xs {
		l.Mean += x / float64(len(xs))
	}
	return l, nil
}
