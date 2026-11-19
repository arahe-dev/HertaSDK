package t4

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrTimeout means the request timed out; a charge MAY have been applied.
var ErrTimeout = errors.New("request timed out")

// ErrUnavailable means the service rejected the request before applying it.
var ErrUnavailable = errors.New("service unavailable")

// ErrDeclined is a permanent rejection.
var ErrDeclined = errors.New("declined")

type Receipt struct {
	User  string
	Cents uint64
	ID    int
}
type Unit struct{}

// Configure hooks before starting requests. Hooks may be called concurrently.
// Zero-value Backend is ready to use. Call Snapshot after requests finish.
type Backend struct {
	ChargeError                                 func(call int) error
	ReserveError                                func(call int) error
	RefundError                                 func(orderID int64, call int) error
	RefundDelay                                 time.Duration
	ChargeWait                                  func(context.Context) error
	RefundEntered                               func(int64)
	RefundRelease                               <-chan struct{}
	mu                                          sync.Mutex
	charges, chargeCalls, reserveCalls, refunds int
	active                                      map[int64]int
	refundOverlap                               bool
	maxRefundActive, refundActive               int
	balances                                    map[string]int64
	debitCalls, creditCalls                     int
}
type Snapshot struct {
	Charges, ChargeCalls, ReserveCalls, Refunds int
	RefundOverlap                               bool
	MaxRefundActive                             int
	DebitCalls, CreditCalls                     int
	Balances                                    map[string]int64
}

func (b *Backend) Snapshot() Snapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	balances := map[string]int64{}
	for k, v := range b.balances {
		balances[k] = v
	}
	return Snapshot{b.charges, b.chargeCalls, b.reserveCalls, b.refunds, b.refundOverlap, b.maxRefundActive, b.debitCalls, b.creditCalls, balances}
}
func (b *Backend) SetBalance(user string, cents int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.balances == nil {
		b.balances = map[string]int64{}
	}
	b.balances[user] = cents
}
func (b *Backend) Charge(ctx context.Context, user string, cents uint64) (Receipt, error) {
	b.mu.Lock()
	b.chargeCalls++
	call := b.chargeCalls
	b.mu.Unlock()
	if b.ChargeWait != nil {
		if err := b.ChargeWait(ctx); err != nil {
			return Receipt{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	var err error
	if b.ChargeError != nil {
		err = b.ChargeError(call)
	}
	if err != nil && !errors.Is(err, ErrTimeout) {
		return Receipt{}, err
	}
	b.mu.Lock()
	b.charges++
	id := b.charges
	b.mu.Unlock()
	return Receipt{user, cents, id}, err
}

// Reserve is idempotent for a user/SKU pair.
func (b *Backend) Reserve(ctx context.Context, user, sku string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	b.reserveCalls++
	call := b.reserveCalls
	b.mu.Unlock()
	if b.ReserveError != nil {
		return b.ReserveError(call)
	}
	return nil
}
func (b *Backend) Refund(ctx context.Context, orderID int64, cents uint64) error {
	b.mu.Lock()
	if b.active == nil {
		b.active = map[int64]int{}
	}
	b.active[orderID]++
	b.refundActive++
	if b.active[orderID] > 1 {
		b.refundOverlap = true
	}
	if b.refundActive > b.maxRefundActive {
		b.maxRefundActive = b.refundActive
	}
	b.refunds++
	call := b.refunds
	b.mu.Unlock()
	defer func() { b.mu.Lock(); b.active[orderID]--; b.refundActive--; b.mu.Unlock() }()
	if b.RefundEntered != nil {
		b.RefundEntered(orderID)
	}
	if b.RefundRelease != nil {
		select {
		case <-b.RefundRelease:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if b.RefundDelay > 0 {
		select {
		case <-time.After(b.RefundDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if b.RefundError != nil {
		return b.RefundError(orderID, call)
	}
	return ctx.Err()
}
func (b *Backend) Debit(ctx context.Context, user string, cents uint64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.balances == nil {
		b.balances = map[string]int64{}
	}
	if b.balances[user] < int64(cents) {
		return ErrDeclined
	}
	b.balances[user] -= int64(cents)
	b.debitCalls++
	return nil
}
func (b *Backend) Credit(ctx context.Context, user string, cents uint64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.balances == nil {
		b.balances = map[string]int64{}
	}
	b.balances[user] += int64(cents)
	b.creditCalls++
	return nil
}
