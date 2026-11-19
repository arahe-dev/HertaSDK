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
	if s.reserve, e = NewMainReserve(rt); e != nil {
		return nil, e
	}
	if s.checkout, e = NewMainCheckout(rt); e != nil {
		return nil, e
	}
	return s, nil
}
func (s *Service) Charge(ctx context.Context, user string, cents uint64) (Receipt, error) {
	return s.charge.Do(ctx, MainChargeInput{Svc: s, User: user, Cents: cents})
}

// classifyReserve leaves context errors untagged so an expired attempt is
// promoted to Uncertain; Reserve is idempotent, so that is retryable.
func classifyReserve(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return err
	}
	return classify(err)
}

func (s *Service) reserveHost(ctx context.Context, user, sku string) error {
	return classifyReserve(s.b.Reserve(ctx, user, sku))
}

// checkoutHost runs inside the checkout op, which holds no key or units and
// never retries, so the nested calls cannot deadlock or repeat a charge.
func (s *Service) checkoutHost(ctx context.Context, user, sku string, cents uint64) (Receipt, error) {
	if _, e := s.reserve.Do(ctx, MainReserveInput{Svc: s, User: user, Sku: sku}); e != nil {
		return Receipt{}, e
	}
	return s.Charge(ctx, user, cents)
}

func (s *Service) Checkout(ctx context.Context, user, sku string, cents uint64) (Receipt, error) {
	return s.checkout.Do(ctx, MainCheckoutInput{Svc: s, User: user, Sku: sku, Cents: cents})
}
