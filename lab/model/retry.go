package model

import (
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// Rules selects the semantics a model reproduces.
type Rules struct {
	// Fixed applies the CONTRACT §8 deltas. False reproduces v0.2.0.
	Fixed bool
}

// Exec models the retry loop of one call (§5) from the moment it holds its
// resources. Virtual time starts at 0 when Do is called.
type Exec struct {
	rules Rules
	op    Op
	call  Call
	in    int

	now        time.Duration
	canceledAt time.Duration // < 0 while not cancelled
	attStart   time.Duration
	held       bool
	released   bool // holds no longer block once the call is released

	Calls, Retried int
	Done           bool
	Res            Result
}

// NewExec prepares the call's retry loop; in is the call's input.
func NewExec(r Rules, op Op, c Call, in int) *Exec {
	x := &Exec{rules: r, op: op, call: c, in: in, canceledAt: -1}
	if c.Precancel {
		x.canceledAt = 0
	}
	return x
}

// Now is the virtual time elapsed since Do was called.
func (x *Exec) Now() time.Duration { return x.now }

// Run executes attempts until the call finishes or an attempt holds.
func (x *Exec) Run() {
	for !x.Done && !x.held {
		if c := x.callerErr(); c != impl.ClassNil { // §5 step 1, §5.2
			x.finish(Result{Class: c, Outcome: impl.Permanent})
			return
		}
		if x.Calls > 0 {
			x.Retried++
		}
		x.Calls++
		x.attStart = x.now
		a := x.call.Script.At(x.Calls)
		switch a.Block {
		case BlockHold, BlockHoldCtx:
			if !x.released {
				x.held = true
				return
			}
		case BlockDeadline:
			x.now = x.deadlineEnd()
		case BlockSleep:
			x.now += a.Sleep
		}
		x.end(a, false)
	}
}

// Release ends a held attempt; the handler returns its scripted result.
func (x *Exec) Release() {
	x.held, x.released = false, true
	x.end(x.call.Script.At(x.Calls), false)
	x.Run()
}

// Cancel cancels the caller's context. A BlockHoldCtx attempt ends at once.
func (x *Exec) Cancel() {
	if x.canceledAt < 0 {
		x.canceledAt = x.now
	}
	if x.held && x.call.Script.At(x.Calls).Block == BlockHoldCtx {
		x.held = false
		x.end(x.call.Script.At(x.Calls), true)
		x.Run()
	}
}

// CallerErr is the class of the caller context's error (ClassNil if live).
func (x *Exec) CallerErr() impl.Class { return x.callerErr() }

// handlerErr is a handler's error in neutral terms.
type handlerErr struct {
	class   impl.Class
	tagged  bool
	outcome impl.Outcome
}

func (e handlerErr) outcomeOf() impl.Outcome {
	if e.tagged {
		return e.outcome
	}
	return impl.Permanent
}

// end finishes attempt a; ctxEnded marks a hold cut short by the context.
func (x *Exec) end(a Attempt, ctxEnded bool) {
	if a.Cancel && x.canceledAt < 0 {
		x.canceledAt = x.now
	}
	attErr, _ := x.attemptErr()
	var err handlerErr
	switch {
	case ctxEnded:
		err = handlerErr{class: attErr}
	case a.Return == ReturnPanic:
		x.finish(Result{Panic: true})
		return
	case a.Return == ReturnPlain:
		err = handlerErr{class: impl.ClassOther}
	case a.Return == ReturnTag:
		err = handlerErr{class: impl.ClassOther, tagged: true, outcome: failOutcome(a.Outcome)}
	case a.Return == ReturnCtxErr && attErr != impl.ClassNil:
		err = handlerErr{class: attErr}
	default: // ReturnOK, or ReturnCtxErr on a live context
		x.finish(Result{Class: impl.ClassNil, Outcome: impl.Success, Value: x.in})
		return
	}
	x.decide(err)
}

// decide applies the classifier and retry gate to a failed attempt.
func (x *Exec) decide(err handlerErr) {
	attErr, own := x.attemptErr()
	var oc impl.Outcome
	var promote bool
	switch {
	case err.tagged:
		oc = err.outcome
	case x.rules.Fixed:
		// §5 step 4: only the attempt's own deadline, with the caller still
		// live, promotes; one verdict drives both retry and report (Δ3, Δ4).
		oc = impl.Permanent
		if err.class != impl.ClassCanceled && own && x.callerErr() == impl.ClassNil {
			oc, promote = impl.Uncertain, true
		}
	default:
		// v0.2: classify() promotes only DeadlineExceeded, but finish()
		// promotes any untagged error once the attempt context expired, for
		// any reason, the parent's deadline included.
		expired := x.op.Timeout > 0 && attErr == impl.ClassDeadline
		if expired && err.class == impl.ClassDeadline {
			oc = impl.Uncertain
		} else {
			oc = impl.Permanent
		}
		promote = expired
	}
	if x.Calls < x.op.attempts() && x.op.retriesOn(oc) && safeRetry(x.op.Effect, oc) {
		return // Run continues with the next attempt
	}
	r := Result{Class: err.class, Outcome: err.outcomeOf()}
	if promote {
		r.Outcome = impl.Uncertain
	}
	x.finish(r)
}

func (x *Exec) finish(r Result) {
	r.Calls = x.Calls
	x.Res, x.Done, x.held = r, true, false
}

// callerErr reports how the caller's context ended by now. A cancel issued
// at the same instant as the deadline is ordered after it.
func (x *Exec) callerErr() impl.Class {
	c, _ := x.callerEnd()
	return c
}

func (x *Exec) callerEnd() (impl.Class, time.Duration) {
	dl := x.call.Deadline
	expired := dl > 0 && x.now >= dl
	switch {
	case x.canceledAt >= 0 && (!expired || x.canceledAt < dl):
		return impl.ClassCanceled, x.canceledAt
	case expired:
		return impl.ClassDeadline, dl
	}
	return impl.ClassNil, 0
}

// attemptErr is the attempt context's error by now, and whether it was the
// attempt's own timer (rather than the caller) that ended it.
func (x *Exec) attemptErr() (impl.Class, bool) {
	ownAt := x.attStart + x.op.Timeout
	ownFired := x.op.Timeout > 0 && x.now >= ownAt
	c, at := x.callerEnd()
	if ownFired && (c == impl.ClassNil || at > ownAt || (at == ownAt && c == impl.ClassCanceled)) {
		return impl.ClassDeadline, true
	}
	return c, false
}

// deadlineEnd is when an attempt awaiting its context's end wakes up.
func (x *Exec) deadlineEnd() time.Duration {
	end := time.Duration(-1)
	if x.op.Timeout > 0 {
		end = x.attStart + x.op.Timeout
	}
	if dl := x.call.Deadline; dl > 0 && (end < 0 || dl < end) {
		end = dl
	}
	if end < 0 {
		panic("model: attempt awaits a deadline that never comes")
	}
	return end
}

// failOutcome mirrors Fail's normalization: only vouched-for outcomes keep
// their verdict; Success and out-of-range values become Permanent.
func failOutcome(o impl.Outcome) impl.Outcome {
	switch o {
	case impl.Transient, impl.Permanent, impl.Throttled, impl.Uncertain:
		return o
	}
	return impl.Permanent
}

// safeRetry is the Effect gate of §5 step 5.
func safeRetry(e impl.Effect, o impl.Outcome) bool {
	switch o {
	case impl.Transient:
		return true
	case impl.Uncertain:
		return e != impl.NonIdempotent
	}
	return false
}
