package impl

import (
	"context"
	"errors"

	h "github.com/arahe-dev/hertasdk"
)

// Orig is HertaSDK v0.2.0, the reference implementation.
var Orig Impl = origImpl{}

type origImpl struct{}

func (origImpl) Name() string { return "orig" }

func (origImpl) NewRuntime(specs ...Spec) (Runtime, error) {
	ss := make([]h.ResourceSpec, len(specs))
	for i, s := range specs {
		ss[i] = h.ResourceSpec{Name: s.Name, Capacity: s.Capacity}
	}
	rt, err := h.NewRuntime(ss...)
	if err != nil {
		return nil, err
	}
	return origRuntime{rt}, nil
}

func (origImpl) Fail(o Outcome, err error) error { return h.Fail(h.Outcome(o), err) }
func (origImpl) OutcomeOf(err error) Outcome     { return Outcome(h.OutcomeOf(err)) }

func (origImpl) Classify(err error) Class {
	switch {
	case err == nil:
		return ClassNil
	case errors.Is(err, h.ErrOverloaded):
		return ClassOverloaded
	case errors.Is(err, h.ErrRuntimeStopping):
		return ClassStopping
	case errors.Is(err, h.ErrUnsafeRetry):
		return ClassUnsafeRetry
	case errors.Is(err, h.ErrEffectRequired):
		return ClassEffectRequired
	case errors.Is(err, h.ErrInvalidEffect):
		return ClassInvalidEffect
	case errors.Is(err, h.ErrInvalidRetryPolicy):
		return ClassInvalidRetryPolicy
	case errors.Is(err, context.Canceled):
		return ClassCanceled
	case errors.Is(err, context.DeadlineExceeded):
		return ClassDeadline
	}
	return ClassOther
}

type origRuntime struct{ rt *h.Runtime }

func (r origRuntime) NewOp(name string, p Policy, fn Handler) (Op, error) {
	var hp h.Policy[int]
	hp.Effect = h.Effect(p.Effect)
	hp.Admission = h.AdmissionMode(p.Admission)
	hp.Timeout = p.Timeout
	hp.SerializeKey = p.Key
	hp.Retry.MaxAttempts = p.MaxAttempts
	if p.On != nil {
		hp.Retry.On = make(map[h.Outcome]bool, len(p.On))
		for o, v := range p.On {
			hp.Retry.On[h.Outcome(o)] = v
		}
	}
	for _, q := range p.Resources {
		hp.Resources = append(hp.Resources, h.Requirement{Name: q.Name, Units: q.Units})
	}
	var hh h.Handler[int, int]
	if fn != nil {
		hh = h.Handler[int, int](fn)
	}
	op, err := h.NewOperation(r.rt, name, hh, hp)
	if err != nil {
		return nil, err
	}
	return op, nil
}

func (r origRuntime) Shutdown(ctx context.Context) error { return r.rt.Shutdown(ctx) }

func (r origRuntime) Stats() Stats {
	s := r.rt.Stats()
	return Stats{Admitted: s.Admitted, Rejected: s.Rejected, Started: s.Started, Finished: s.Finished, Retried: s.Retried}
}
