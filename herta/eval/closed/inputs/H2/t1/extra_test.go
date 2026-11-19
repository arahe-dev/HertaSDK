package t1

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestChargePolicy(t *testing.T) {
	ctx := context.Background()
	b := &Backend{ChargeError: func(c int) error {
		if c < 3 {
			return ErrUnavailable
		}
		return nil
	}}
	s, _ := New(b)
	if _, e := s.Charge(ctx, "a", 1); e != nil || b.Snapshot().ChargeCalls != 3 || b.Snapshot().Charges != 1 {
		t.Fatal(e, b.Snapshot())
	}
	b = &Backend{ChargeError: func(int) error { return ErrTimeout }}
	s, _ = New(b)
	if _, e := s.Charge(ctx, "a", 1); !errors.Is(e, ErrTimeout) || b.Snapshot().ChargeCalls != 1 {
		t.Fatal(e, b.Snapshot())
	}
	b = &Backend{ChargeError: func(int) error { return ErrDeclined }}
	s, _ = New(b)
	if _, e := s.Charge(ctx, "a", 1); !errors.Is(e, ErrDeclined) || b.Snapshot().ChargeCalls != 1 {
		t.Fatal(e)
	}
	b = &Backend{ChargeWait: func(c context.Context) error { <-c.Done(); return c.Err() }}
	s, _ = New(b)
	st := time.Now()
	if _, e := s.Charge(ctx, "a", 1); e == nil || b.Snapshot().ChargeCalls != 1 || time.Since(st) > 900*time.Millisecond {
		t.Fatal(e, b.Snapshot(), time.Since(st))
	}
}
