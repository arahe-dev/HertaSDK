package t1

import (
	"context"
	"errors"

	"github.com/arahe-dev/hertasdk"
)

// classify maps backend errors to outcomes. Only ErrUnavailable is known not
// to have applied the charge, so it is the only retryable (Transient) error.
// Context errors stay untagged so an expired attempt deadline is promoted to
// Uncertain by the runtime, and a caller's own cancellation is returned as is.
func classify(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrUnavailable):
		return hertasdk.Fail(hertasdk.Transient, err)
	case errors.Is(err, ErrTimeout):
		return hertasdk.Fail(hertasdk.Uncertain, err)
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return err
	default:
		return hertasdk.Fail(hertasdk.Permanent, err)
	}
}

type servicePtr = *Service
type Service struct {
	b      *Backend
	charge *hertasdk.Operation[MainChargeInput, Receipt]
}

func (s *Service) chargeHost(ctx context.Context, user string, cents uint64) (Receipt, error) {
	r, err := s.b.Charge(ctx, user, cents)
	if err != nil {
		return Receipt{}, classify(err)
	}
	return r, nil
}

func New(b *Backend) (*Service, error) {
	s := &Service{b: b}
	rt, e := NewMainRuntime()
	if e != nil {
		return nil, e
	}
	if s.charge, e = NewMainCharge(rt); e != nil {
		return nil, e
	}
	return s, nil
}

func (s *Service) Charge(ctx context.Context, user string, cents uint64) (Receipt, error) {
	return s.charge.Do(ctx, MainChargeInput{Svc: s, User: user, Cents: cents})
}
