package t3

import (
	"context"
	"errors"
	"fmt"
	"github.com/arahe-dev/hertasdk"
	"sync"
	"testing"
	"time"
)

var _ = errors.Is
var _ = fmt.Sprint
var _ sync.WaitGroup
var _ = hertasdk.OutcomeOf

func service(t *testing.T, b *Backend) *Service {
	t.Helper()
	s, e := New(b)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func bounded(t *testing.T, f func(context.Context) error) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- f(ctx) }()
	select {
	case e := <-done:
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("operation hung beyond watchdog")
		return nil
	}
}

func TestOracleCheckout(t *testing.T) {
	t.Run("inventory_retry", func(t *testing.T) {
		b := &Backend{ReserveError: func(call int) error {
			if call < 3 {
				return ErrUnavailable
			}
			return nil
		}}
		s := service(t, b)
		r, e := s.Checkout(context.Background(), "u", "sku", 25)
		v := b.Snapshot()
		if e != nil || r.Cents != 25 || v.ReserveCalls != 3 || v.Charges != 1 {
			t.Fatalf("inventory recovery: %v receipt=%+v snapshot=%+v", e, r, v)
		}
	})
	t.Run("ambiguous_charge", func(t *testing.T) {
		b := &Backend{ChargeError: func(call int) error {
			if call == 1 {
				return ErrTimeout
			}
			return nil
		}}
		s := service(t, b)
		_, e := s.Checkout(context.Background(), "u", "sku", 25)
		v := b.Snapshot()
		if !errors.Is(e, ErrTimeout) || v.Charges != 1 || v.ChargeCalls != 1 {
			t.Fatalf("checkout repeated a payment: err=%v charges=%d calls=%d reservations=%d", e, v.Charges, v.ChargeCalls, v.ReserveCalls)
		}
	})
	t.Run("inventory_exhausted", func(t *testing.T) {
		b := &Backend{ReserveError: func(int) error { return ErrUnavailable }}
		s := service(t, b)
		_, e := s.Checkout(context.Background(), "u", "sku", 25)
		v := b.Snapshot()
		if !errors.Is(e, ErrUnavailable) || v.ReserveCalls != 3 || v.Charges != 0 {
			t.Fatalf("failed inventory: %v %+v", e, v)
		}
	})
	t.Run("inventory_declined", func(t *testing.T) {
		b := &Backend{ReserveError: func(int) error { return ErrDeclined }}
		s := service(t, b)
		_, e := s.Checkout(context.Background(), "u", "sku", 25)
		v := b.Snapshot()
		if !errors.Is(e, ErrDeclined) || v.ReserveCalls != 1 || v.Charges != 0 {
			t.Fatalf("permanent inventory failure: %v %+v", e, v)
		}
	})
	t.Run("empty", func(t *testing.T) {
		b := &Backend{}
		s := service(t, b)
		r, e := s.Checkout(context.Background(), "", "", 0)
		if e != nil || r.Cents != 0 || b.Snapshot().Charges != 1 {
			t.Fatalf("empty input: %+v %v", r, e)
		}
	})
}
