package t1

import (
	"context"
	"errors"

	"github.com/arahe-dev/hertasdk"
)

type servicePtr = *Service
type Service struct {
	b      *Backend
	charge *hertasdk.Operation[MainChargeInput, Receipt]
}

func (s *Service) chargeHost(ctx context.Context, user string, cents uint64) (Receipt, error) {
	r, err := s.b.Charge(ctx, user, cents)
	if err == nil {
		return r, nil
	}
	outcome := hertasdk.Permanent
	switch {
	case errors.Is(err, ErrTimeout), errors.Is(err, context.DeadlineExceeded):
		outcome = hertasdk.Uncertain
	case errors.Is(err, ErrDeclined), errors.Is(err, context.Canceled):
	case errors.Is(err, ErrUnavailable):
		outcome = hertasdk.Transient
	}
	return r, &hertasdk.Failure{Outcome: outcome, Err: err}
}
func New(b *Backend) (*Service, error) {
	rt, e := NewMainRuntime()
	if e != nil {
		return nil, e
	}
	charge, e := NewMainCharge(rt)
	if e != nil {
		return nil, e
	}
	return &Service{b: b, charge: charge}, nil
}
func (s *Service) Charge(ctx context.Context, user string, cents uint64) (Receipt, error) {
	return s.charge.Do(ctx, MainChargeInput{Svc: s, User: user, Cents: cents})
}
