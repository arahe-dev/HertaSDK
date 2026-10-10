package hertasdk

import (
	"context"
	"errors"
	"fmt"
)

// Outcome classifies the result of one attempt. The handler or its provider
// adapter classifies; Herta never infers a verdict except the single
// attempt-deadline promotion in classify.
type Outcome int

const (
	Success Outcome = iota
	Transient
	Permanent
	Throttled
	Uncertain
)

func (o Outcome) String() string {
	switch o {
	case Success:
		return "Success"
	case Transient:
		return "Transient"
	case Permanent:
		return "Permanent"
	case Throttled:
		return "Throttled"
	case Uncertain:
		return "Uncertain"
	}
	return fmt.Sprintf("Outcome(%d)", int(o))
}

// Failure tags an error with its Outcome.
type Failure struct {
	Outcome Outcome
	Err     error
}

// Error and Unwrap accept a nil *Failure: a handler that returns one by
// mistake (var f *Failure; return v, f) must not crash its caller.
func (f *Failure) Error() string {
	if f == nil {
		return "hertasdk: <nil failure>"
	}
	if f.Err == nil {
		return fmt.Sprintf("hertasdk: failure(%s)", f.Outcome)
	}
	return f.Err.Error()
}

func (f *Failure) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.Err
}

// tagged returns the first Failure in err's chain. A nil *Failure tags
// nothing: absence of semantics has no semantics, so it reads as untagged.
func tagged(err error) (*Failure, bool) {
	f, ok := errors.AsType[*Failure](err)
	return f, ok && f != nil
}

// Fail tags err with o. Fail(_, nil) is nil. An error that already carries
// a Failure is returned unchanged: the verdict closest to the failure wins.
// Success and out-of-range outcomes are normalized to Permanent, because
// nobody vouched for them.
func Fail(o Outcome, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := tagged(err); ok {
		return err
	}
	switch o {
	case Transient, Permanent, Throttled, Uncertain:
	default:
		o = Permanent
	}
	return &Failure{Outcome: o, Err: err}
}

// OutcomeOf reports err's Outcome: Success for nil, the first Failure in
// the chain if any, Permanent otherwise.
func OutcomeOf(err error) Outcome {
	if err == nil {
		return Success
	}
	if f, ok := tagged(err); ok {
		return f.Outcome
	}
	return Permanent
}

// classify is the one classifier: its verdict drives both the retry
// decision and the error Do returns (Δ3). expired means the attempt's own
// deadline fired while the caller's context was still live, so a caller
// ending its context is never promoted (Δ4). promoted reports that the
// returned error must be tagged Uncertain.
func classify(err error, expired bool) (o Outcome, promoted bool) {
	if f, ok := tagged(err); ok {
		return f.Outcome, false
	}
	if expired && !errors.Is(err, context.Canceled) {
		return Uncertain, true
	}
	return Permanent, false
}

// outcomeSet is an immutable set of Outcomes; the retry table is one.
type outcomeSet uint8

func setOf(os ...Outcome) outcomeSet {
	var s outcomeSet
	for _, o := range os {
		s |= 1 << o
	}
	return s
}

// has is false for every out-of-range Outcome, so a hand-built Failure
// carrying garbage is never retried.
func (s outcomeSet) has(o Outcome) bool { return uint(o) < 8 && s&(1<<o) != 0 }
