package mutate

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"
)

// Triage maps a mutant Key to the reason it is equivalent: no test can
// kill it because the program's observable behavior is unchanged.
type Triage map[string]string

// ReadTriage loads a triage file: one "key<TAB>reason" per line, with
// blank lines and #-comments ignored. A missing file is an empty triage.
func ReadTriage(name string) (Triage, error) {
	t := Triage{}
	f, err := os.Open(name)
	if os.IsNotExist(err) {
		return t, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, why, ok := strings.Cut(line, "\t")
		if !ok || strings.TrimSpace(why) == "" {
			return nil, fmt.Errorf("%s:%d: want key<TAB>reason", name, n)
		}
		t[k] = strings.TrimSpace(why)
	}
	return t, sc.Err()
}

// Apply marks each surviving mutant the triage explains as Equivalent and
// returns the entries for the run's files that matched no survivor: stale
// verdicts to re-check.
// A triaged mutant that was killed is a contradiction and is reported too.
func (t Triage) Apply(rs []Result) (stale []string) {
	used, files := map[string]bool{}, map[string]bool{}
	for i := range rs {
		files[rs[i].File] = true
		why, ok := t[rs[i].Key()]
		if !ok {
			continue
		}
		used[rs[i].Key()] = true
		switch rs[i].Status {
		case Survived:
			rs[i].Status, rs[i].Reason = Equivalent, why
		case Killed:
			stale = append(stale, rs[i].Key()+" (triaged equivalent, but killed)")
		}
	}
	for k := range t {
		if file, _, _ := strings.Cut(k, "|"); files[file] && !used[k] {
			stale = append(stale, k)
		}
	}
	slices.Sort(stale)
	return stale
}
