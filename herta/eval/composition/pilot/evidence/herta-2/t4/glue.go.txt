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
	if s.transfer, e = NewMainTransfer(rt); e != nil {
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

// transferLockPrefix namespaces transfer keys away from the account keys that
// debit and credit lock on the same runtime. Keys are runtime-wide: a transfer
// keyed on the raw account would self-deadlock on its nested debit, and an
// empty account would not be serialized at all.
const transferLockPrefix = "\x00transfer\x00"

func transferLock(from string) string { return transferLockPrefix + from }

// transferHost runs under the source account's transfer key and claims no
// units. It nests only the debit: the account it waits on is always shorter
// than any transfer key it holds, so chains of held keys cannot cycle even when
// account IDs look like transfer keys. Crediting under the key would let a
// destination equal to a held transfer key deadlock (self or ABBA).
func (s *Service) transferHost(ctx context.Context, _, from string, cents uint64) error {
	return s.Debit(ctx, from, cents)
}

// Transfer debits under the per-source transfer key, then credits the
// destination. Credits commute, so serializing the debit side is what orders
// transfers from one account.
func (s *Service) Transfer(ctx context.Context, from, to string, cents uint64) error {
	if _, e := s.transfer.Do(ctx, MainTransferInput{Svc: s, Lock: transferLock(from), From: from, Cents: cents}); e != nil {
		return e
	}
	return s.Credit(ctx, to, cents)
}
