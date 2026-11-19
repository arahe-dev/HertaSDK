package t4

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
	debit    *hertasdk.Operation[MainDebitInput, Unit]
	credit   *hertasdk.Operation[MainCreditInput, Unit]
	transfer *hertasdk.Operation[MainTransferInput, Unit]
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
	s.debit, e = NewMainDebit(rt)
	if e != nil {
		return nil, e
	}
	s.credit, e = NewMainCredit(rt)
	if e != nil {
		return nil, e
	}
	s.transfer, e = NewMainTransfer(rt)
	if e != nil {
		return nil, e
	}
	return s, nil
}
func (s *Service) Debit(ctx context.Context, user string, cents uint64) error {
	_, e := s.debit.Do(ctx, MainDebitInput{Svc: s, User: user, Cents: cents})
	return e
}
func (s *Service) Credit(ctx context.Context, user string, cents uint64) error {
	_, e := s.credit.Do(ctx, MainCreditInput{Svc: s, User: user, Cents: cents})
	return e
}
func (s *Service) Transfer(ctx context.Context, from, to string, cents uint64) error {
	_, e := s.transfer.Do(ctx, MainTransferInput{Svc: s, From: from, To: to, Cents: cents, TransferKey: "transfer:" + from})
	return e
}
