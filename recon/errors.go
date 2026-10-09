package execution

import (
	"context"
	"errors"
	"runtime"
)

// Construction errors: an unsafe contract fails in NewOperation, never
// under load.
var (
	// ErrUnsafeRetry: the retry table could duplicate a side effect
	// (NonIdempotent with On[Uncertain]).
	ErrUnsafeRetry = errors.New("execution: unsafe retry contract")
	// ErrEffectRequired: Policy.Effect was left as EffectUnknown.
	ErrEffectRequired = errors.New("execution: effect must be declared (Pure, Idempotent, or NonIdempotent)")
	// ErrInvalidEffect: Policy.Effect is out of range.
	ErrInvalidEffect = errors.New("execution: invalid effect value")
	// ErrInvalidRetryPolicy: Retry.On asks for Success, Permanent,
	// Throttled or an out-of-range Outcome.
	ErrInvalidRetryPolicy = errors.New("execution: invalid retry policy")
	// ErrInvalidOutcome is kept for API parity with v0.2; Fail normalizes
	// instead of returning it.
	ErrInvalidOutcome = errors.New("execution: invalid outcome")
)

// Admission errors returned by Do.
var (
	// ErrOverloaded is always returned as Fail(Throttled, ErrOverloaded).
	ErrOverloaded = errors.New("execution: overloaded")
	// ErrRuntimeStopping is returned untagged (Permanent) once Shutdown has
	// begun: new calls are refused and blocked waiters are withdrawn.
	ErrRuntimeStopping = errors.New("execution: runtime is stopping")
)

// refused is the error for a call whose key or units were not granted.
// Reject reports overload; a withdrawn waiter reports the caller's own
// context error first (Δ6), then shutdown.
func (rt *Runtime) refused(ctx context.Context, mode AdmissionMode) error {
	if mode == Reject {
		rt.stats.rejected.Add(1)
		// Yield once: a caller that retries a refusal at once is waiting,
		// and must not hoard the CPU the holders need to finish
		// (CONTRACT §4.1; lab/results/r1/reject-probe/ROOT-CAUSE.md).
		runtime.Gosched()
		return &Failure{Outcome: Throttled, Err: ErrOverloaded}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrRuntimeStopping
}

// failed is the error for a call whose last attempt failed: the handler's
// error, tagged Uncertain when classify promoted it. Every Do failure after
// admission is built here, so a typed error carrying the operation name,
// Outcome and attempt count replaces this one function.
func failed(err error, promoted bool) error {
	if promoted {
		return &Failure{Outcome: Uncertain, Err: err}
	}
	return err
}
