package model

import "github.com/arahe-dev/hertasdk/lab/impl"

// ValidateRuntime predicts NewRuntime's error class (§3.1, plus v0.2's
// refusal of an empty spec list).
func ValidateRuntime(specs []impl.Spec) impl.Class {
	if len(specs) == 0 {
		return impl.ClassOther
	}
	seen := map[string]bool{}
	for _, s := range specs {
		if s.Name == "" || s.Capacity <= 0 || seen[s.Name] {
			return impl.ClassOther
		}
		seen[s.Name] = true
	}
	return impl.ClassNil
}

// Validate predicts NewOperation's error class (§3.2), in v0.2's check
// order: handler, name, Effect, resources, retry table, retry safety.
func Validate(specs []impl.Spec, name string, p impl.Policy, nilHandler bool) impl.Class {
	if nilHandler || name == "" {
		return impl.ClassOther
	}
	switch p.Effect {
	case impl.EffectUnknown:
		return impl.ClassEffectRequired
	case impl.Pure, impl.Idempotent, impl.NonIdempotent:
	default:
		return impl.ClassInvalidEffect
	}
	caps := map[string]int64{}
	for _, s := range specs {
		caps[s.Name] = s.Capacity
	}
	seen := map[string]bool{}
	for _, r := range p.Resources {
		c, known := caps[r.Name]
		if seen[r.Name] || !known || r.Units <= 0 || r.Units > c {
			return impl.ClassOther
		}
		seen[r.Name] = true
	}
	for o, want := range p.On {
		if want && o != impl.Transient && o != impl.Uncertain {
			return impl.ClassInvalidRetryPolicy
		}
	}
	if p.Effect == impl.NonIdempotent && p.On[impl.Uncertain] {
		return impl.ClassUnsafeRetry
	}
	return impl.ClassNil
}
