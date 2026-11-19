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

type servicePtr = *Service
type Service struct {
	b        *Backend
	debit    *hertasdk.Operation[MainDebitInput, Unit]
	credit   *hertasdk.Operation[MainCreditInput, Unit]
	transfer *hertasdk.Operation[MainTransferInput, Unit]
}

func userKey(user string) string     { return "user:" + user }
func transferKey(from string) string { return "transfer:" + from }

func (s *Service) transferHost(ctx context.Context, from, to string, cents uint64) error {
	if e := s.Debit(ctx, from, cents); e != nil {
		return e
	}
	return s.Credit(ctx, to, cents)
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
	_, e := s.debit.Do(ctx, MainDebitInput{Svc: s, User: user, UserKey: userKey(user), Cents: cents})
	return e
}
func (s *Service) Credit(ctx context.Context, user string, cents uint64) error {
	_, e := s.credit.Do(ctx, MainCreditInput{Svc: s, User: user, UserKey: userKey(user), Cents: cents})
	return e
}
func (s *Service) Transfer(ctx context.Context, from, to string, cents uint64) error {
	_, e := s.transfer.Do(ctx, MainTransferInput{Svc: s, From: from, To: to, FromKey: transferKey(from), Cents: cents})
	return e
}
