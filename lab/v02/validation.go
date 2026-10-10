package execution

import (
	"errors"
	"fmt"
)

// Construction-time validation errors. Invalid contracts must fail in
// NewRuntime/NewOperation — never in production request paths.
var (
	// ErrUnsafeRetry is returned by NewOperation when a Policy's retry
	// semantics could duplicate side effects (e.g. NonIdempotent +
	// retry on Uncertain).
	ErrUnsafeRetry = errors.New("execution: unsafe retry contract")
	// ErrEffectRequired is returned by NewOperation when the Policy's
	// Effect is the zero value (EffectUnknown): a Policy that forgot to
	// declare its semantic class must fail at construction, not silently
	// claim the strongest retry-safety semantics.
	ErrEffectRequired = errors.New("execution: effect must be declared (Pure, Idempotent, or NonIdempotent)")
	// ErrInvalidEffect is returned by NewOperation for an out-of-range
	// Effect value (not one of EffectUnknown/Pure/Idempotent/NonIdempotent).
	ErrInvalidEffect = errors.New("execution: invalid effect value")
	// ErrInvalidRetryPolicy is returned by NewOperation when the retry
	// policy requests behavior the V0 contract does not implement —
	// e.g. On[Throttled] = true (Herta has no backoff/jitter, so a
	// throttled retry would only hammer the refusing resource), or On
	// entries for outcomes that can never be retried (Success,
	// Permanent, or invalid outcome values).
	ErrInvalidRetryPolicy = errors.New("execution: invalid retry policy")
	// ErrInvalidOutcome is returned by Fail when asked to manufacture a
	// Failure carrying an invalid Outcome (out-of-range, or Success with
	// a non-nil error — a "successful failure" is a caller bug).
	ErrInvalidOutcome = errors.New("execution: invalid outcome")
	// ErrOverloaded is returned by Do when Reject admission finds no
	// capacity. Always wrapped: Fail(Throttled, ErrOverloaded) — so both
	// errors.Is(err, ErrOverloaded) and OutcomeOf(err) == Throttled work.
	ErrOverloaded = errors.New("execution: overloaded")
	// ErrRuntimeStopping is returned by Do (and admission waiters) once
	// Shutdown has begun: new work is rejected, waiting work is withdrawn.
	ErrRuntimeStopping = errors.New("execution: runtime is stopping")
)

// validateRuntime is applied by NewRuntime (see runtime.go).

// validateOperation checks a Policy against the runtime it will run on.
func validateOperation[I any](rt *Runtime, name string, p Policy[I]) error {
	if name == "" {
		return errors.New("execution: operation name must not be empty")
	}

	// Effect is a declared contract, not an implied default. The zero
	// value is EffectUnknown — omission is a construction error, never a
	// silent claim of Pure retry-safety.
	switch p.Effect {
	case EffectUnknown:
		return fmt.Errorf("%w: operation %q", ErrEffectRequired, name)
	case Pure, Idempotent, NonIdempotent:
		// Declared: proceed.
	default:
		return fmt.Errorf("%w: operation %q has Effect %d", ErrInvalidEffect, name, int(p.Effect))
	}

	seen := map[string]bool{}
	for _, req := range p.Resources {
		if seen[req.Name] {
			return fmt.Errorf("execution: operation %q lists resource %q twice", name, req.Name)
		}
		seen[req.Name] = true

		res, ok := rt.resources[req.Name]
		if !ok {
			return fmt.Errorf("execution: operation %q requires unknown resource %q", name, req.Name)
		}
		if req.Units <= 0 {
			return fmt.Errorf("execution: operation %q resource %q units must be > 0, got %d", name, req.Name, req.Units)
		}
		if req.Units > res.capacity {
			return fmt.Errorf("execution: operation %q requires %d units of %s, capacity is %d",
				name, req.Units, res.name, res.capacity)
		}
	}

	// RetryPolicy.On must only request behavior the V0 contract actually
	// implements. Policies the executor would silently ignore are
	// rejected instead — silence is how contracts rot.
	for o, want := range p.Retry.On {
		if !want {
			continue // false entries are inert and legal
		}
		switch o {
		case Throttled:
			// Observable but not automatically retryable in V0: Herta has
			// no backoff/jitter, so a throttled retry would only hammer
			// the resource that just refused us.
			return fmt.Errorf("%w: operation %q requests retry on Throttled; Throttled is observable but not retryable in V0 (no backoff semantics)", ErrInvalidRetryPolicy, name)
		case Success, Permanent:
			// Can never be retried by contract; asking for it is a bug.
			return fmt.Errorf("%w: operation %q requests retry on %s, which is never retryable", ErrInvalidRetryPolicy, name, o)
		case Transient, Uncertain:
			// The actionable V0 entries (Uncertain still gated by the
			// Effect safety check below and isSafeRetry).
		default:
			// Unrecognized Outcome value in the map.
			return fmt.Errorf("%w: operation %q has an invalid Outcome value (%d) in Retry.On", ErrInvalidRetryPolicy, name, int(o))
		}
	}

	if p.Effect == NonIdempotent && p.Retry.mayRetry(Uncertain) {
		return fmt.Errorf(
			"%w: operation %q is non-idempotent and cannot automatically retry uncertain outcomes",
			ErrUnsafeRetry, name,
		)
	}
	return nil
}

// isSafeRetry is the single source of truth for "which (Effect, Outcome)
// combinations may auto-retry". Called from the Do retry loop as a
// belt-and-braces double-check alongside construction validation.
func isSafeRetry(e Effect, o Outcome) bool {
	switch o {
	case Success, Permanent, Throttled:
		// Success: nothing to retry. Permanent: will never succeed by
		// repeating. Throttled: NOT retried in V0 — Herta has no
		// backoff/jitter semantics, so an immediate retry would only
		// hammer the resource that just refused us (construction rejects
		// On[Throttled] = true for the same reason).
		return false
	case Transient:
		// Transient failure of any Effect is safe to retry: the attempt
		// did not reach the external system (adapter's responsibility to
		// classify correctly).
		return true
	case Uncertain:
		// The attempt MIGHT have taken effect: only idempotent or pure
		// operations may repeat it.
		return e != NonIdempotent
	default:
		return false
	}
}
