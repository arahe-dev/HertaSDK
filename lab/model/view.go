package model

import (
	"fmt"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// Phase is where a call is, as a black-box observer sees it.
type Phase int

const (
	Idle    Phase = iota // not started
	Parked               // blocked inside Do, outside the handler
	Holding              // inside a held handler attempt
	Done                 // Do returned (or panicked)
)

func (p Phase) String() string { return [...]string{"idle", "parked", "holding", "done"}[p] }

// ShutdownView is one Shutdown call.
type ShutdownView struct {
	Returned bool
	Class    impl.Class // its error, once returned
	Either   bool       // model only: nil and ctx.Err() are both v0.2 behavior
	Lag      bool       // observed only: Finished < Admitted when it returned nil
}

// View is everything observable once a step has settled.
type View struct {
	Phases    []Phase
	Results   []Result // meaningful where Phases[i] == Done
	Stats     impl.Stats
	Shutdowns []ShutdownView
}

// Diff explains the first way got departs from the model's want, or "".
// Under v0.2 rules a Finished lag at Shutdown is tolerated (Δ2).
func Diff(r Rules, want, got View) string {
	for i := range want.Phases {
		w, g := want.Phases[i], got.Phases[i]
		if w != g {
			return fmt.Sprintf("c%d: phase %v, model says %v", i, g, w)
		}
		if w == Done && want.Results[i] != got.Results[i] {
			return fmt.Sprintf("c%d: result %v, model says %v", i, got.Results[i], want.Results[i])
		}
	}
	if want.Stats != got.Stats {
		return fmt.Sprintf("stats %+v, model says %+v", got.Stats, want.Stats)
	}
	if len(want.Shutdowns) != len(got.Shutdowns) {
		return fmt.Sprintf("%d Shutdown calls, model says %d", len(got.Shutdowns), len(want.Shutdowns))
	}
	for i, w := range want.Shutdowns {
		g := got.Shutdowns[i]
		switch {
		case w.Returned != g.Returned:
			return fmt.Sprintf("Shutdown #%d returned=%v, model says %v", i, g.Returned, w.Returned)
		case w.Returned && w.Class != g.Class && !(w.Either && g.Class == impl.ClassCanceled):
			return fmt.Sprintf("Shutdown #%d returned %v, model says %v", i, g.Class, w.Class)
		case g.Lag && r.Fixed:
			return fmt.Sprintf("Shutdown #%d returned before Finished caught up (Δ2)", i)
		}
	}
	return ""
}
