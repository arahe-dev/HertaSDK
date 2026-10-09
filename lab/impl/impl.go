// Package impl is the lab's neutral view of a Herta implementation.
//
// Every suite (assumptions, models, systems, DoE) is written against these
// types, then run against each registered implementation. This lets v0.2.0
// (Orig) and v0.2-recon (Recon) be compared by the same code, and v0.3+ be
// added later with one adapter file.
//
// The adapter costs one interface call and one func conversion per Do. That
// is equal for every implementation, so it cancels out in comparisons. Pure
// micro-benchmarks should use the packages directly.
package impl

import (
	"context"
	"time"
)

// Outcome, Effect and Admission use the same numbering as the
// implementations.
type Outcome int

const (
	Success Outcome = iota
	Transient
	Permanent
	Throttled
	Uncertain
)

// Outcomes lists every valid Outcome, in order.
var Outcomes = []Outcome{Success, Transient, Permanent, Throttled, Uncertain}

type Effect int

const (
	EffectUnknown Effect = iota
	Pure
	Idempotent
	NonIdempotent
)

// Effects lists every declarable Effect.
var Effects = []Effect{Pure, Idempotent, NonIdempotent}

type Admission int

const (
	Wait Admission = iota
	Reject
)

// Spec is a resource budget.
type Spec struct {
	Name     string
	Capacity int64
}

// Req is one resource demand of an operation.
type Req struct {
	Name  string
	Units int64
}

// Policy is the neutral form of Policy[int] (Retry is flattened in).
type Policy struct {
	Effect      Effect
	Resources   []Req
	Admission   Admission
	Timeout     time.Duration
	Key         func(int) string
	MaxAttempts int
	On          map[Outcome]bool
}

// Handler is the neutral handler shape: int in, int out.
type Handler func(ctx context.Context, in int) (int, error)

// Stats mirrors the implementation's Stats.
type Stats struct {
	Admitted, Rejected, Started, Finished, Retried uint64
}

// Class is an implementation-independent classification of an error, so
// suites can compare errors across implementations whose sentinels differ.
type Class int

const (
	ClassNil Class = iota
	ClassOverloaded
	ClassStopping
	ClassUnsafeRetry
	ClassEffectRequired
	ClassInvalidEffect
	ClassInvalidRetryPolicy
	ClassCanceled
	ClassDeadline
	ClassOther
)

func (c Class) String() string {
	return [...]string{"nil", "overloaded", "stopping", "unsafe-retry", "effect-required",
		"invalid-effect", "invalid-retry-policy", "canceled", "deadline", "other"}[c]
}

// Impl is one Herta implementation.
type Impl interface {
	Name() string
	NewRuntime(specs ...Spec) (Runtime, error)
	// Fail and OutcomeOf use the implementation's own tagging.
	Fail(o Outcome, err error) error
	OutcomeOf(err error) Outcome
	// Classify maps err onto Class using the implementation's sentinels.
	Classify(err error) Class
}

type Runtime interface {
	NewOp(name string, p Policy, h Handler) (Op, error)
	Shutdown(ctx context.Context) error
	Stats() Stats
}

type Op interface {
	Do(ctx context.Context, in int) (int, error)
}

// All returns every registered implementation; Orig is always first.
func All() []Impl { return []Impl{Orig, Recon} }

// ByName returns the implementation named n, or nil.
func ByName(n string) Impl {
	for _, i := range All() {
		if i.Name() == n {
			return i
		}
	}
	return nil
}
