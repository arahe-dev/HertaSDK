package execution

import (
	"context"
	"time"
)

// API parity with v0.2.0 (CONTRACT §2), checked at compile time: every
// exported identifier with its exact shape.
var (
	_ func(...ResourceSpec) (*Runtime, error)                                              = NewRuntime
	_ func(*Runtime, context.Context) error                                                = (*Runtime).Shutdown
	_ func(*Runtime) Stats                                                                 = (*Runtime).Stats
	_ func(*Runtime, string, Handler[int, int], Policy[int]) (*Operation[int, int], error) = NewOperation[int, int]
	_ func(*Operation[int, int], context.Context, int) (int, error)                        = (*Operation[int, int]).Do
	_ func(Outcome, error) error                                                           = Fail
	_ func(error) Outcome                                                                  = OutcomeOf
	_ func(*Resource) string                                                               = (*Resource).String
	_ func(Effect) string                                                                  = Effect.String
	_ func(Outcome) string                                                                 = Outcome.String
	_ error                                                                                = (*Failure)(nil)
	_ func(*Failure) error                                                                 = (*Failure).Unwrap
	_ Handler[int, int]                                                                    = func(context.Context, int) (int, error) { return 0, nil }
	_ *KeyLocker

	_ = ResourceSpec{Name: "", Capacity: int64(0)}
	_ = Requirement{Name: "", Units: int64(0)}
	_ = RetryPolicy{MaxAttempts: 0, On: map[Outcome]bool{}}
	_ = Failure{Outcome: Success, Err: error(nil)}
	_ = Stats{Admitted: uint64(0), Rejected: 0, Started: 0, Finished: 0, Retried: 0}
	_ = Policy[int]{Effect: EffectUnknown, Resources: []Requirement{}, Admission: Wait,
		Timeout: time.Duration(0), SerializeKey: func(int) string { return "" }, Retry: RetryPolicy{}}

	_ = [...]Effect{EffectUnknown: EffectUnknown, Pure: Pure, Idempotent: Idempotent, NonIdempotent: NonIdempotent}
	_ = [...]AdmissionMode{Wait: Wait, Reject: Reject}
	_ = [...]Outcome{Success: Success, Transient: Transient, Permanent: Permanent, Throttled: Throttled, Uncertain: Uncertain}
	_ = [...]error{ErrUnsafeRetry, ErrEffectRequired, ErrInvalidEffect, ErrInvalidRetryPolicy,
		ErrInvalidOutcome, ErrOverloaded, ErrRuntimeStopping}
)
