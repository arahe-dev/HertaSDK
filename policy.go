package execution

import (
	"context"
	"time"
)

// Handler is a plain Go function wrapped by an Operation.
type Handler[I any, O any] func(context.Context, I) (O, error)

// Effect declares an operation's semantic class — the minimum information
// Herta needs to say "this retry policy is unsafe".
type Effect int

const (
	// Pure: calculation/transformation; no external side effects.
	Pure Effect = iota
	// Idempotent: repeating is safe (write keyed by unique event_id,
	// PUT-like update, same-brand catalogue replace).
	Idempotent
	// NonIdempotent: repeating may duplicate a side effect (external
	// provider call, send message, charge customer).
	NonIdempotent
)

// Requirement is one resource demand: `Units` of the named Resource.
// Units must be > 0 and <= capacity (validated at construction).
type Requirement struct {
	Name  string
	Units int64
}

// Policy is the boring, declarative contract of an Operation.
type Policy[I any] struct {
	// Effect declares the operation's semantic class (validated against Retry).
	Effect Effect

	// Resources lists the runtime budgets this operation consumes per attempt.
	Resources []Requirement

	// Admission controls overload behavior: Wait (block for capacity) or
	// Reject (fail fast with ErrOverloaded). Queue comes later.
	Admission AdmissionMode

	// Timeout is the COOPERATIVE per-attempt deadline propagated through
	// context.Context. It does NOT forcibly terminate handlers: a handler
	// that ignores its context can run longer. Worst case ≈
	// MaxAttempts × Timeout. Zero means no deadline.
	Timeout time.Duration

	// SerializeKey optionally derives a serialization key from the input.
	// Operations sharing a key execute one-at-a-time (key acquired BEFORE
	// resources, so a queued same-key request holds zero resource budget).
	// A nil function or an empty derived key means "no serialization".
	SerializeKey func(I) string

	// Retry classifies which Outcomes may be retried. Semantically unsafe
	// combinations are rejected at construction (e.g. NonIdempotent +
	// On[Uncertain]).
	Retry RetryPolicy
}

// AdmissionMode selects between blocking and fail-fast overload behavior.
type AdmissionMode int

const (
	// Wait blocks until capacity is available, the caller's context is
	// done, or runtime shutdown begins.
	Wait AdmissionMode = iota
	// Reject fails immediately with ErrOverloaded (Throttled) when the
	// requested capacity is not immediately available.
	Reject
)

// Queue admission is descoped from V0: the V0 contract is exactly
// Wait | Reject (see the hertasdk-go README descope note). A
// bounded-queue mode changes the execution model (queue-depth accounting,
// overflow outcomes, head-of-line behavior) and is not added without a
// real consumer that needs it.

// RetryPolicy: MaxAttempts is the TOTAL number of attempts.
// 0 → 1 attempt. 1 → 1 attempt. N → at most N attempts. No backoff in V0.
type RetryPolicy struct {
	MaxAttempts int
	// On maps Outcome → "may retry". Only Transient/Throttled/Uncertain
	// are meaningful; Success/Permanent never retry.
	On map[Outcome]bool
}

// attempts normalizes MaxAttempts per the frozen semantics.
func (r RetryPolicy) attempts() int {
	if r.MaxAttempts <= 1 {
		return 1
	}
	return r.MaxAttempts
}

// mayRetry reports whether the outcome is retryable under this policy.
func (r RetryPolicy) mayRetry(o Outcome) bool {
	if r.On == nil {
		return false
	}
	return r.On[o]
}
