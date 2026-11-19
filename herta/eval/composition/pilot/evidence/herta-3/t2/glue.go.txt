package t2

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
	refund *hertasdk.Operation[MainRefundInput, Unit]
}

func New(b *Backend) (*Service, error) {
	s := &Service{b: b}
	rt, e := NewMainRuntime()
	if e != nil {
		return nil, e
	}
	s.refund, e = NewMainRefund(rt)
	if e != nil {
		return nil, e
	}
	return s, nil
}
func (s *Service) Refund(ctx context.Context, orderID int64, cents uint64) error {
	_, e := s.refund.Do(ctx, MainRefundInput{
		Svc:      s,
		OrderId:  orderID,
		OrderKey: strconv.FormatInt(orderID, 10),
		Cents:    cents,
	})
	return e
}
