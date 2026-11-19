package t3

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
	case errors.Is(err, ErrUnavailable):
		return hertasdk.Fail(hertasdk.Transient, err)
	case errors.Is(err, ErrTimeout):
		return hertasdk.Fail(hertasdk.Uncertain, err)
	default:
		return hertasdk.Fail(hertasdk.Permanent, err)
	}
}

type servicePtr = *Service
type Service struct {
	b        *Backend
	charge   *hertasdk.Operation[MainChargeInput, Receipt]
	reserve  *hertasdk.Operation[MainReserveInput, Unit]
	checkout *hertasdk.Operation[MainCheckoutInput, Receipt]
}

func (s *Service) chargeHost(ctx context.Context, user string, cents uint64) (Receipt, error) {
	r, e := s.b.Charge(ctx, user, cents)
	return r, classify(e)
}

// classifyReserve maps inventory errors to outcomes. Context errors stay
// untagged so caller cancellation is reported as-is and never retried.
func classifyReserve(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, ErrTimeout):
		return hertasdk.Fail(hertasdk.Uncertain, err)
	case errors.Is(err, ErrUnavailable):
		return hertasdk.Fail(hertasdk.Transient, err)
	default:
		return hertasdk.Fail(hertasdk.Permanent, err)
	}
}

func (s *Service) reserveHost(ctx context.Context, user, sku string) error {
	return classifyReserve(s.b.Reserve(ctx, user, sku))
}

func (s *Service) checkoutHost(ctx context.Context, user, sku string, cents uint64) (Receipt, error) {
	if _, e := s.reserve.Do(ctx, MainReserveInput{Svc: s, User: user, Sku: sku}); e != nil {
		return Receipt{}, e
	}
	return s.charge.Do(ctx, MainChargeInput{Svc: s, User: user, Cents: cents})
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
	s.reserve, e = NewMainReserve(rt)
	if e != nil {
		return nil, e
	}
	s.checkout, e = NewMainCheckout(rt)
	if e != nil {
		return nil, e
	}
	return s, nil
}
func (s *Service) Charge(ctx context.Context, user string, cents uint64) (Receipt, error) {
	return s.charge.Do(ctx, MainChargeInput{Svc: s, User: user, Cents: cents})
}
func (s *Service) Checkout(ctx context.Context, user, sku string, cents uint64) (Receipt, error) {
	return s.checkout.Do(ctx, MainCheckoutInput{Svc: s, User: user, Sku: sku, Cents: cents})
}
