package t1

import (
	"context"
	"errors"
	"github.com/arahe-dev/hertasdk"
	"strconv"
	"time"
)

var _ = time.Second
var _ = strconv.FormatInt
var _ = errors.Is

func classify(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrTimeout):
		return hertasdk.Fail(hertasdk.Uncertain, err)
	case errors.Is(err, ErrUnavailable):
		return hertasdk.Fail(hertasdk.Transient, err)
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
	r, e := s.b.Charge(ctx, user, cents)
	return r, classify(e)
}
func New(b *Backend) (*Service, error) {
	s := &Service{b: b}
	rt, e := NewMainRuntime()
	if e != nil {
		return nil, e
	}
	_ = rt
	s.charge, e = NewMainCharge(rt)
	if e != nil {
		return nil, e
	}
	return s, nil
}
func (s *Service) Charge(ctx context.Context, user string, cents uint64) (Receipt, error) {
	return s.charge.Do(ctx, MainChargeInput{Svc: s, User: user, Cents: cents})
}
