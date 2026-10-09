package mutate

// Tally counts verdicts over a set of results.
type Tally struct {
	Total, Compile, Killed, Survived, Equivalent int
	ByStage                                      map[string]int // kills per stage
}

func (t *Tally) add(r Result) {
	t.Total++
	switch r.Status {
	case Compile:
		t.Compile++
	case Killed:
		t.Killed++
		if t.ByStage == nil {
			t.ByStage = map[string]int{}
		}
		t.ByStage[r.Stage]++
	case Survived:
		t.Survived++
	case Equivalent:
		t.Equivalent++
	}
}

// Score is the mutation score: killed over killable (equivalents excluded).
func (t Tally) Score() float64 { return ratio(t.Killed, t.Killed+t.Survived) }

// Raw is the score before triage: equivalents count as survivors.
func (t Tally) Raw() float64 { return ratio(t.Killed, t.Killed+t.Survived+t.Equivalent) }

// StageScore is the score had the cascade stopped after stages.
func (t Tally) StageScore(stages ...string) float64 {
	k := 0
	for _, s := range stages {
		k += t.ByStage[s]
	}
	return ratio(k, t.Killed+t.Survived)
}

// Tallies totals rs overall and per operator.
func Tallies(rs []Result) (all Tally, by map[Operator]Tally) {
	by = map[Operator]Tally{}
	for _, r := range rs {
		all.add(r)
		t := by[r.Op]
		t.add(r)
		by[r.Op] = t
	}
	return all, by
}

// ratio is n/d, or -1 when d is 0 (rendered as "n/a").
func ratio(n, d int) float64 {
	if d == 0 {
		return -1
	}
	return float64(n) / float64(d)
}
