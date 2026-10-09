package execution

import (
	"context"
	"fmt"
	"time"
)

// Handler is ordinary business logic. Herta never inspects its result.
type Handler[I any, O any] func(context.Context, I) (O, error)

// Effect declares what repeating an attempt does to the outside world.
// The zero value is EffectUnknown and is rejected at construction: absence
// of semantics never grants retry safety.
type Effect int

const (
	EffectUnknown Effect = iota
	Pure
	Idempotent
	NonIdempotent
)

func (e Effect) String() string {
	switch e {
	case EffectUnknown:
		return "EffectUnknown"
	case Pure:
		return "Pure"
	case Idempotent:
		return "Idempotent"
	case NonIdempotent:
		return "NonIdempotent"
	}
	return fmt.Sprintf("Effect(%d)", int(e))
}

// retrySafe is the set of outcomes that may be repeated under e. Transient
// is safe for every Effect: the adapter vouches the attempt did not run.
func (e Effect) retrySafe() outcomeSet {
	if e == NonIdempotent {
		return setOf(Transient)
	}
	return setOf(Transient, Uncertain)
}

// AdmissionMode selects what a call does when its key or units are taken.
// Any value other than Reject behaves as Wait, the conservative zero value.
type AdmissionMode int

const (
	// Wait blocks until granted, the caller's context ends, or Shutdown
	// begins.
	Wait AdmissionMode = iota
	// Reject fails at once with Fail(Throttled, ErrOverloaded).
	Reject
)

// Requirement claims Units of the named resource for each call.
type Requirement struct {
	Name  string
	Units int64
}

// RetryPolicy: MaxAttempts is the total number of attempts (≤ 1 means 1).
// On lists the outcomes that may be retried; only Transient and Uncertain
// are legal, and On is copied at construction.
type RetryPolicy struct {
	MaxAttempts int
	On          map[Outcome]bool
}

// Policy is an Operation's declarative contract. NewOperation validates and
// snapshots it; later mutation of the value, its slice or its map has no
// effect.
type Policy[I any] struct {
	Effect    Effect
	Resources []Requirement
	Admission AdmissionMode
	// Timeout is a cooperative per-attempt deadline; ≤ 0 means none.
	Timeout time.Duration
	// SerializeKey derives a key; calls sharing a non-empty key run one at
	// a time, and a call waiting for its key holds no resource units.
	SerializeKey func(I) string
	Retry        RetryPolicy
}

// ResourceSpec declares a named finite capacity.
type ResourceSpec struct {
	Name     string
	Capacity int64
}
