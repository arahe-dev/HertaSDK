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
	rt, e := NewMainRuntime()
	if e != nil {
		return nil, e
	}
	refund, e := NewMainRefund(rt)
	if e != nil {
		return nil, e
	}
	return &Service{b: b, refund: refund}, nil
}
func (s *Service) Refund(ctx context.Context, orderID int64, cents uint64) error {
	_, e := s.refund.Do(ctx, MainRefundInput{
		Svc: s, OrderId: orderID, Cents: cents, OrderKey: strconv.FormatInt(orderID, 10),
	})
	return e
}
