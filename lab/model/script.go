package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// Op declares one operation. Keyed ops serialize on the per-call Key.
type Op struct {
	Name        string
	Effect      impl.Effect
	Admission   impl.Admission
	Resources   []impl.Req
	Keyed       bool
	Timeout     time.Duration
	MaxAttempts int
	On          []impl.Outcome
}

// Policy renders the op as a neutral policy; key derives the call's key.
func (o Op) Policy(key func(int) string) impl.Policy {
	p := impl.Policy{Effect: o.Effect, Admission: o.Admission, Resources: o.Resources,
		Timeout: o.Timeout, MaxAttempts: o.MaxAttempts}
	if o.Keyed {
		p.Key = key
	}
	if o.On != nil {
		p.On = map[impl.Outcome]bool{}
		for _, oc := range o.On {
			p.On[oc] = true
		}
	}
	return p
}

func (o Op) attempts() int { return max(o.MaxAttempts, 1) }

func (o Op) retriesOn(oc impl.Outcome) bool {
	for _, x := range o.On {
		if x == oc {
			return true
		}
	}
	return false
}

// Call is one Do invocation. Its input is its index in the scenario.
type Call struct {
	Op        int
	Key       string // "" means unserialized, even on a keyed op
	Precancel bool   // the caller's context is cancelled before Do
	Deadline  time.Duration
	Script    Script
}

// Block is how a handler attempt waits before it returns.
type Block int

const (
	BlockNone     Block = iota
	BlockHold           // until the call is released; ignores ctx
	BlockHoldCtx        // until released, or ctx ends (then returns ctx.Err())
	BlockDeadline       // until the attempt context ends
	BlockSleep          // for Attempt.Sleep, ignoring ctx
)

// Return is what a handler attempt returns once it stops blocking.
type Return int

const (
	ReturnOK     Return = iota // (input, nil)
	ReturnPlain                // ErrScripted, untagged
	ReturnCtxErr               // (input, ctx.Err())
	ReturnTag                  // Fail(Outcome, ErrScripted)
	ReturnPanic                // panic(ErrScripted)
)

// Attempt scripts one handler invocation.
type Attempt struct {
	Block   Block
	Sleep   time.Duration
	Cancel  bool // cancel the caller's context just before returning
	Return  Return
	Outcome impl.Outcome
}

// Script is a call's handler behavior per attempt; the last entry repeats.
type Script []Attempt

// At returns the script for attempt n (1-based).
func (s Script) At(n int) Attempt {
	if len(s) == 0 {
		return Attempt{}
	}
	return s[min(n, len(s))-1]
}

// Result is the caller-visible result of one Do, in neutral terms.
type Result struct {
	Class   impl.Class
	Outcome impl.Outcome // OutcomeOf(err)
	Value   int
	Panic   bool
	Calls   int // handler invocations
}

func (r Result) String() string {
	if r.Panic {
		return fmt.Sprintf("panic after %d calls", r.Calls)
	}
	return fmt.Sprintf("{%s %s v=%d calls=%d}", r.Class, outcomeName(r.Outcome), r.Value, r.Calls)
}

func outcomeName(o impl.Outcome) string {
	names := []string{"Success", "Transient", "Permanent", "Throttled", "Uncertain"}
	if o >= 0 && int(o) < len(names) {
		return names[o]
	}
	return fmt.Sprintf("Outcome(%d)", int(o))
}

func (a Attempt) String() string {
	var b strings.Builder
	b.WriteString([...]string{"", "hold ", "hold-ctx ", "await-deadline ", "sleep "}[a.Block])
	if a.Block == BlockSleep {
		fmt.Fprintf(&b, "%v ", a.Sleep)
	}
	if a.Cancel {
		b.WriteString("cancel-caller ")
	}
	switch a.Return {
	case ReturnTag:
		fmt.Fprintf(&b, "tag(%s)", outcomeName(a.Outcome))
	default:
		b.WriteString([...]string{"ok", "plain", "ctx-err", "", "panic"}[a.Return])
	}
	return b.String()
}

func (o Op) String() string {
	on := make([]string, len(o.On))
	for i, x := range o.On {
		on[i] = outcomeName(x)
	}
	effect := [...]string{"EffectUnknown", "Pure", "Idempotent", "NonIdempotent"}
	return fmt.Sprintf("%s %s admission=%s resources=%v keyed=%v timeout=%v attempts=%d on=%v",
		o.Name, effect[o.Effect], [...]string{"Wait", "Reject"}[o.Admission], o.Resources, o.Keyed, o.Timeout, o.MaxAttempts, on)
}
