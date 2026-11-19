package t2

import (
	"context"
	"strconv"

	"github.com/arahe-dev/hertasdk"
)

type servicePtr = *Service
type Service struct {
	b      *Backend
	refund *hertasdk.Operation[MainRefundInput, Unit]
}

func New(b *Backend) (*Service, error) {
	s := &Service{b: b}
	rt, e := NewMainRuntime()
	if e != nil {
		return nil, e
	}
	if s.refund, e = NewMainRefund(rt); e != nil {
		return nil, e
	}
	return s, nil
}

// orderKey is never empty (an empty key would disable serialization) and is
// injective over all int64 values, including 0 and negatives.
func orderKey(orderID int64) string { return "order:" + strconv.FormatInt(orderID, 10) }

// refundHost runs while the refund op holds orderKey for this order.
func (s *Service) refundHost(ctx context.Context, _ string, orderID int64, cents uint64) error {
	return s.b.Refund(ctx, orderID, cents)
}

func (s *Service) Refund(ctx context.Context, orderID int64, cents uint64) error {
	_, e := s.refund.Do(ctx, MainRefundInput{Svc: s, OrderKey: orderKey(orderID), OrderId: orderID, Cents: cents})
	return e
}
