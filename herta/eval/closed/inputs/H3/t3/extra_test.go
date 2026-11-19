package t3

import (
	"context"
	"errors"
	"testing"
)

func TestReserveRetried(t *testing.T) {
	for _, err := range []error{ErrUnavailable, ErrTimeout} {
		b := &Backend{ReserveError: func(c int) error {
			if c < 3 {
				return err
			}
			return nil
		}}
		s, _ := New(b)
		r, e := s.Checkout(context.Background(), "a", "x", 3)
		v := b.Snapshot()
		if e != nil || r.Cents != 3 || v.ReserveCalls != 3 || v.Charges != 1 {
			t.Fatalf("%v %+v", e, v)
		}
	}
}

func TestReserveFailsNoCharge(t *testing.T) {
	for _, err := range []error{ErrUnavailable, ErrDeclined} {
		b := &Backend{ReserveError: func(int) error { return err }}
		s, _ := New(b)
		_, e := s.Checkout(context.Background(), "a", "x", 3)
		v := b.Snapshot()
		if !errors.Is(e, err) || v.ChargeCalls != 0 {
			t.Fatalf("%v %+v", e, v)
		}
	}
}

func TestChargeTimeoutOnce(t *testing.T) {
	b := &Backend{ChargeError: func(int) error { return ErrTimeout }}
	s, _ := New(b)
	_, e := s.Checkout(context.Background(), "a", "x", 3)
	v := b.Snapshot()
	if !errors.Is(e, ErrTimeout) || v.ChargeCalls != 1 || v.Charges != 1 {
		t.Fatalf("%v %+v", e, v)
	}
}

func TestChargeUnavailableRetried(t *testing.T) {
	b := &Backend{ChargeError: func(c int) error {
		if c == 1 {
			return ErrUnavailable
		}
		return nil
	}}
	s, _ := New(b)
	_, e := s.Checkout(context.Background(), "a", "x", 3)
	v := b.Snapshot()
	if e != nil || v.ChargeCalls != 2 || v.Charges != 1 || v.ReserveCalls != 1 {
		t.Fatalf("%v %+v", e, v)
	}
}
