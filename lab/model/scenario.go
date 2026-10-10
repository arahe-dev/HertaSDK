package model

import (
	"errors"
	"fmt"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// Scenario is a reproducible workload: resources, operations, calls, and
// the order in which calls are started, released and cancelled.
type Scenario struct {
	Specs []impl.Spec
	Ops   []Op
	Calls []Call
	Steps []Step
}

// StepKind is one externally triggered event.
type StepKind int

const (
	StepStart    StepKind = iota // call Do for Step.Call
	StepRelease                  // let Step.Call's held attempt return
	StepCancel                   // cancel Step.Call's context
	StepShutdown                 // call Shutdown (Expired: with a done context)
)

// Step is one event. After each step every goroutine has settled.
type Step struct {
	Kind    StepKind
	Call    int
	Expired bool
}

func (s Step) String() string {
	switch s.Kind {
	case StepShutdown:
		if s.Expired {
			return "Shutdown(expired ctx)"
		}
		return "Shutdown"
	}
	return fmt.Sprintf("%s(c%d)", [...]string{"Start", "Release", "Cancel"}[s.Kind], s.Call)
}

// ErrAmbiguous marks a step whose outcome depends on goroutine scheduling
// (CONTRACT §8: nondeterministic in v0.2). Generators skip such steps.
var ErrAmbiguous = errors.New("model: step outcome depends on scheduling")

// KeyRejectCounted records the reading of §7 used for Δ1: a Reject call
// refused because its key is held is a "Reject overload" and counts in
// Rejected.
const KeyRejectCounted = true
