package execution

import (
	"context"
	"fmt"
	"time"
)

// Handler is a plain Go function wrapped by an Operation.
type Handler[I any, O any] func(context.Context, I) (O, error)

// Effect declares an operation's semantic class — the minimum information
// Herta needs to say "this retry policy is unsafe".
//
// The zero value is EffectUnknown, NOT Pure: omission is a contract error,
// not silent consent. If semantics matter, absence of semantics must not
// itself have semantics — a Policy without an explicit Effect fails at
// construction (NewOperation), never at request time.
type Effect int

const (
	// EffectUnknown is the zero value of Effect. It is rejected at
	// construction: a Policy that forgot to declare its semantic class
	// must not default to the strongest retry-safety claim (Pure).
	EffectUnknown Effect = iota
	// Pure: calculation/transformation; no external side effects.
	Pure
	// Idempotent: repeating is safe (write keyed by unique event_id,
	// PUT-like update, same-brand catalogue replace).
	Idempotent
	// NonIdempotent: repeating may duplicate a side effect (external
	// provider call, send message, charge customer).
	NonIdempotent
)

// String renders the effect name (also makes %s formatting valid).
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
	default:
		return fmt.Sprintf("Effect(%d)", int(e))
	}
}

// valid reports whether e is a declared semantic class.
func (e Effect) valid() bool {
	switch e {
	case Pure, Idempotent, NonIdempotent:
		return true
	default:
		return false
	}
}

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
	// done, or runtime shutdown begins. Wait is deliberately the zero
	// value: the conservative default is to block, never to shed.
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
//
// On maps Outcome → "may retry". In V0 the only actionable entries are:
//
//   - Transient: always retryable when the Effect permits the retry to be
//     safe (isSafeRetry).
//   - Uncertain: actionable only for Pure/Idempotent operations.
//
// Throttled is OBSERVABLE but NOT automatically retryable in V0 — Herta has
// no backoff/jitter semantics, so a throttled retry would only hammer the
// resource that just refused us. On[Throttled] = true is rejected at
// construction rather than silently ignored. Success/Permanent are never
// retryable regardless of On.
type RetryPolicy struct {
	MaxAttempts int
	// On maps Outcome → "may retry" (see the V0 actionability rules
	// above).
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
