package execution

import (
	"context"
	"errors"
	"fmt"
)

// Outcome classifies the result of one attempt. Classification is the
// handler/provider-adapter's responsibility — the code closest to the
// external system knows whether a failure is Transient or Uncertain; Herta
// never pretends to infer it.
type Outcome int

const (
	Success Outcome = iota
	Transient
	Permanent
	Throttled
	Uncertain
)

// String renders the outcome name (also makes %s formatting valid).
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
	default:
		return fmt.Sprintf("Outcome(%d)", int(o))
	}
}

// Failure wraps an error with its Outcome so retry decisions and callers
// can branch on semantics, not error strings.
type Failure struct {
	Outcome Outcome
	Err     error
}

// Error implements error.
func (f *Failure) Error() string {
	if f.Err == nil {
		return fmt.Sprintf("execution: failure(%s)", f.Outcome)
	}
	return f.Err.Error()
}

// Unwrap lets errors.Is/As see through the wrapper.
func (f *Failure) Unwrap() error { return f.Err }

// Fail tags err with an Outcome. Fail(_, nil) returns nil so handlers can
// write `return execution.Fail(...)` unconditionally.
func Fail(o Outcome, err error) error {
	if err == nil {
		return nil
	}
	// Re-wrapping an already-classified error keeps the innermost verdict
	// (the adapter closest to the failure knows best).
	var f *Failure
	if errors.As(err, &f) {
		return err
	}
	return &Failure{Outcome: o, Err: err}
}

// OutcomeOf returns the Outcome recorded for err (Permanent for untagged
// errors, Success for nil). One lookup, no competing error models.
func OutcomeOf(err error) Outcome {
	if err == nil {
		return Success
	}
	var f *Failure
	if errors.As(err, &f) {
		return f.Outcome
	}
	return Permanent
}

// classify maps a raw attempt error to an Outcome.
//
// Rules (frozen for V0):
//
//   - nil error                  → Success
//   - *Failure                   → its Outcome (innermost wins; see Fail)
//   - plain error                → Permanent  (nobody vouched for it; never retry)
//   - context.DeadlineExceeded/
//     context.Canceled after the
//     attempt deadline expired   → Uncertain (conservative promotion: the
//     deadline fired mid-attempt, so side-effect state is unknown unless
//     the adapter classified it differently)
//   - caller-cancelled context   → Canceled, no promotion (caller gave up;
//     there is nothing to retry)
func classify(err error, deadlineExpired bool) Outcome {
	if err == nil {
		return Success
	}
	var f *Failure
	if errors.As(err, &f) {
		return f.Outcome
	}
	if errors.Is(err, context.Canceled) {
		return Permanent
	}
	if errors.Is(err, context.DeadlineExceeded) && deadlineExpired {
		return Uncertain
	}
	return Permanent
}
