package t3

import (
	"context"
	"errors"
	"testing"
)

func TestCheckoutPolicy(t *testing.T) {
	ctx := context.Background()
	b := &Backend{ReserveError: func(c int) error {
		if c == 1 {
			return ErrUnavailable
		}
		if c == 2 {
			return ErrTimeout
		}
		return nil
	}}
	s, _ := New(b)
	if _, e := s.Checkout(ctx, "a", "x", 1); e != nil || b.Snapshot().ReserveCalls != 3 || b.Snapshot().Charges != 1 {
		t.Fatal(e, b.Snapshot())
	}
	b = &Backend{ReserveError: func(int) error { return ErrUnavailable }}
	s, _ = New(b)
	if _, e := s.Checkout(ctx, "a", "x", 1); !errors.Is(e, ErrUnavailable) || b.Snapshot().ReserveCalls != 3 || b.Snapshot().ChargeCalls != 0 {
		t.Fatal(e, b.Snapshot())
	}
	b = &Backend{ChargeError: func(int) error { return ErrTimeout }}
	s, _ = New(b)
	if _, e := s.Checkout(ctx, "a", "x", 1); !errors.Is(e, ErrTimeout) || b.Snapshot().ChargeCalls != 1 {
		t.Fatal(e, b.Snapshot())
	}
}
