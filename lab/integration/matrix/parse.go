package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// event is one line of go test -json (cmd/test2json).
type event struct {
	Action      string
	Package     string
	Test        string
	Elapsed     float64
	Output      string
	OutputType  string
	FailedBuild string
}

// tally counts one test's (or package's) results across -count runs.
type tally struct{ pass, fail, skip int }

func (t tally) String() string {
	switch {
	case t == tally{}:
		return "absent"
	case t.fail > 0:
		return fmtCounts("FAIL", t)
	case t.pass > 0:
		return fmtCounts("pass", t)
	}
	return "skip"
}

func fmtCounts(word string, t tally) string {
	if t.pass+t.fail+t.skip == 1 {
		return word
	}
	return fmt.Sprintf("%s %d/%d", word, t.pass, t.pass+t.fail+t.skip)
}

// pkg is one package's results.
type pkg struct {
	result  string // ok, FAIL, build-fail or no-tests
	elapsed float64
	tests   map[string]*tally
	why     map[string]string // a skipped test's last log line: its reason
}

// run is a whole go test -json stream, keyed by import path.
type run map[string]*pkg

func parse(r io.Reader) (run, error) {
	out := run{}
	get := func(name string) *pkg {
		p := out[name]
		if p == nil {
			p = &pkg{tests: map[string]*tally{}, why: map[string]string{}}
			out[name] = p
		}
		return p
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var e event
		if !strings.HasPrefix(sc.Text(), "{") {
			continue // stray build output
		}
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return nil, err
		}
		if e.Package == "" {
			continue
		}
		p := get(e.Package)
		switch {
		case e.Test == "" && e.Action == "pass":
			p.result, p.elapsed = "ok", p.elapsed+e.Elapsed
		case e.Test == "" && e.Action == "fail":
			p.result, p.elapsed = "FAIL", p.elapsed+e.Elapsed
			if e.FailedBuild != "" {
				p.result = "build-fail"
			}
		case e.Test == "" && e.Action == "skip":
			p.result = "no-tests"
		case e.Test != "" && e.Action == "output" && e.OutputType == "":
			if s := strings.TrimSpace(e.Output); s != "" {
				p.why[e.Test] = s
			}
		case e.Test != "" && (e.Action == "pass" || e.Action == "fail" || e.Action == "skip"):
			t := p.tests[e.Test]
			if t == nil {
				t = &tally{}
				p.tests[e.Test] = t
			}
			switch e.Action {
			case "pass":
				t.pass++
			case "fail":
				t.fail++
			case "skip":
				t.skip++
			}
		}
	}
	return out, sc.Err()
}

func (p *pkg) counts() (pass, fail, skip int) {
	for _, t := range p.tests {
		switch {
		case t.fail > 0:
			fail++
		case t.pass > 0:
			pass++
		default:
			skip++
		}
	}
	return
}
