package t4

import (
	"context"

	"github.com/arahe-dev/hertasdk"
)

type servicePtr = *Service
type Service struct {
	b         *Backend
	debit     *hertasdk.Operation[MainDebitInput, Unit]
	credit    *hertasdk.Operation[MainCreditInput, Unit]
	transfer  *hertasdk.Operation[MainTransferInput, Unit]
	moveFunds *hertasdk.Operation[MainMoveFundsInput, Unit]
}

func New(b *Backend) (*Service, error) {
	s := &Service{b: b}
	rt, e := NewMainRuntime()
	if e != nil {
		return nil, e
	}
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
	s.moveFunds, e = NewMainMoveFunds(rt)
	if e != nil {
		return nil, e
	}
	return s, nil
}
func (s *Service) Debit(ctx context.Context, user string, cents uint64) error {
	_, e := s.debit.Do(ctx, MainDebitInput{Svc: s, User: user, Cents: cents, AccountKey: "account:" + user})
	return e
}
func (s *Service) Credit(ctx context.Context, user string, cents uint64) error {
	_, e := s.credit.Do(ctx, MainCreditInput{Svc: s, User: user, Cents: cents, AccountKey: "account:" + user})
	return e
}
func (s *Service) Transfer(ctx context.Context, from, to string, cents uint64) error {
	first, second := from, to
	if second < first {
		first, second = second, first
	}
	firstKey := "transfer:" + first
	secondKey := ""
	if first != second {
		secondKey = "transfer:" + second
	}
	_, err := s.transfer.Do(ctx, MainTransferInput{
		Svc: s, From: from, To: to, Cents: cents, FirstKey: firstKey, SecondKey: secondKey,
	})
	return err
}
func (s *Service) transferHost(ctx context.Context, from, to string, cents uint64, secondKey string) error {
	_, err := s.moveFunds.Do(ctx, MainMoveFundsInput{Svc: s, From: from, To: to, Cents: cents, SecondKey: secondKey})
	return err
}
func (s *Service) moveFundsHost(ctx context.Context, from, to string, cents uint64) error {
	if err := s.Debit(ctx, from, cents); err != nil {
		return err
	}
	return s.Credit(ctx, to, cents)
}
