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
		// repeating. Throttled: safe but pointless to hammer; capacity is
		// the problem, not the failure.
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
