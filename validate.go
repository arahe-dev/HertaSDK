package hertasdk

import (
	"errors"
	"fmt"
)

// Validation runs in the same order and reports the same text as v0.2.
// Where v0.2 depended on map iteration order, the lowest Outcome decides.

func validateSpecs(specs []ResourceSpec) error {
	if len(specs) == 0 {
		return errors.New("hertasdk: runtime requires at least one resource")
	}
	seen := make(map[string]bool, len(specs))
	for _, s := range specs {
		if s.Name == "" {
			return errors.New("hertasdk: resource name must not be empty")
		}
		if s.Capacity <= 0 {
			return fmt.Errorf("hertasdk: resource %q capacity must be > 0, got %d", s.Name, s.Capacity)
		}
		if seen[s.Name] {
			return fmt.Errorf("hertasdk: duplicate resource %q", s.Name)
		}
		seen[s.Name] = true
	}
	return nil
}

func validateEffect(name string, e Effect) error {
	switch e {
	case Pure, Idempotent, NonIdempotent:
		return nil
	case EffectUnknown:
		return fmt.Errorf("%w: operation %q", ErrEffectRequired, name)
	}
	return fmt.Errorf("%w: operation %q has Effect %d", ErrInvalidEffect, name, int(e))
}

// resolveClaims checks each requirement in the caller's order and resolves
// it to its budget.
func resolveClaims(rt *Runtime, name string, reqs []Requirement) ([]claim, error) {
	claims := make([]claim, 0, len(reqs))
	seen := make(map[string]bool, len(reqs))
	for _, req := range reqs {
		if seen[req.Name] {
			return nil, fmt.Errorf("hertasdk: operation %q lists resource %q twice", name, req.Name)
		}
		seen[req.Name] = true
		r, ok := rt.budgets[req.Name]
		if !ok {
			return nil, fmt.Errorf("hertasdk: operation %q requires unknown resource %q", name, req.Name)
		}
		if req.Units <= 0 {
			return nil, fmt.Errorf("hertasdk: operation %q resource %q units must be > 0, got %d", name, req.Name, req.Units)
		}
		if req.Units > r.size {
			return nil, fmt.Errorf("hertasdk: operation %q requires %d units of %s, capacity is %d",
				name, req.Units, r.name, r.size)
		}
		claims = append(claims, claim{r: r, n: req.Units})
	}
	return claims, nil
}

// retryTable compiles Retry.On into an immutable set (Δ5), restricted to
// the outcomes e makes safe to repeat.
func retryTable(name string, on map[Outcome]bool, e Effect) (outcomeSet, error) {
	var set outcomeSet
	bad, found := Outcome(0), false
	for o, want := range on {
		switch {
		case !want:
		case o == Transient || o == Uncertain:
			set |= setOf(o)
		case !found || o < bad:
			bad, found = o, true
		}
	}
	if found {
		return 0, invalidRetry(name, bad)
	}
	if e == NonIdempotent && set.has(Uncertain) {
		return 0, fmt.Errorf("%w: operation %q is non-idempotent and cannot automatically retry uncertain outcomes",
			ErrUnsafeRetry, name)
	}
	return set & e.retrySafe(), nil
}

func invalidRetry(name string, o Outcome) error {
	switch o {
	case Throttled:
		return fmt.Errorf("%w: operation %q requests retry on Throttled; Throttled is observable but not retryable in V0 (no backoff semantics)",
			ErrInvalidRetryPolicy, name)
	case Success, Permanent:
		return fmt.Errorf("%w: operation %q requests retry on %s, which is never retryable",
			ErrInvalidRetryPolicy, name, o)
	}
	return fmt.Errorf("%w: operation %q has an invalid Outcome value (%d) in Retry.On",
		ErrInvalidRetryPolicy, name, int(o))
}
