package t1

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/arahe-dev/hertasdk"
)

var _ = time.Second
var _ = strconv.FormatInt
var _ = errors.Is

type servicePtr = *Service
type Service struct {
	b      *Backend
	charge *hertasdk.Operation[MainChargeInput, Receipt]
}

// classifyCharge maps the provider's error contract to hertasdk outcomes.
// ErrTimeout is checked first: a timed-out charge may have been applied, so
// it must never be reported (or retried) as Transient. Context errors stay
// untagged so the runtime can promote an attempt-deadline expiry to
// Uncertain and report caller cancellation as-is.
func classifyCharge(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrTimeout):
		return hertasdk.Fail(hertasdk.Uncertain, err)
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, ErrUnavailable):
		return hertasdk.Fail(hertasdk.Transient, err)
	default:
		return hertasdk.Fail(hertasdk.Permanent, err)
	}
}

func (s *Service) chargeHost(ctx context.Context, user string, cents uint64) (Receipt, error) {
	r, e := s.b.Charge(ctx, user, cents)
	if e != nil {
		return Receipt{}, classifyCharge(e)
	}
	return r, nil
}
func New(b *Backend) (*Service, error) {
	s := &Service{b: b}
	rt, e := NewMainRuntime()
	if e != nil {
		return nil, e
	}
	s.charge, e = NewMainCharge(rt)
	if e != nil {
		return nil, e
	}
	return s, nil
}
func (s *Service) Charge(ctx context.Context, user string, cents uint64) (Receipt, error) {
	return s.charge.Do(ctx, MainChargeInput{Svc: s, User: user, Cents: cents})
}
