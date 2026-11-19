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

var _ = fmt.Sprint
var _ sync.WaitGroup
var _ = time.Second
var _ = hertasdk.OutcomeOf

func TestOracleHardCheckout(t *testing.T) {
	t.Run("uncertain_inventory", func(t *testing.T) {
		b := &Backend{ReserveError: func(call int) error {
			if call < 3 {
				return fmt.Errorf("inventory: %w", ErrTimeout)
			}
			return nil
		}}
		s := service(t, b)
		_, e := s.Checkout(context.Background(), "u", "sku", 10)
		v := b.Snapshot()
		if e != nil || v.ReserveCalls != 3 || v.Charges != 1 || v.ChargeCalls != 1 {
			t.Fatalf("idempotent inventory recovery: %v %+v", e, v)
		}
	})
	t.Run("uncertain_inventory_exhausted", func(t *testing.T) {
		b := &Backend{ReserveError: func(int) error { return ErrTimeout }}
		s := service(t, b)
		_, e := s.Checkout(context.Background(), "u", "sku", 10)
		v := b.Snapshot()
		if !errors.Is(e, ErrTimeout) || v.ReserveCalls != 3 || v.Charges != 0 {
			t.Fatalf("idempotent inventory limit: %v %+v", e, v)
		}
	})
	t.Run("declined_charge", func(t *testing.T) {
		b := &Backend{ChargeError: func(int) error { return ErrDeclined }}
		s := service(t, b)
		_, e := s.Checkout(context.Background(), "u", "sku", 10)
		v := b.Snapshot()
		if !errors.Is(e, ErrDeclined) || v.ReserveCalls != 1 || v.ChargeCalls != 1 || v.Charges != 0 {
			t.Fatalf("declined charge repeated: %v %+v", e, v)
		}
	})
	t.Run("concurrent_capacity", func(t *testing.T) {
		b := &Backend{}
		s := service(t, b)
		e := bounded(t, func(ctx context.Context) error {
			var wg sync.WaitGroup
			errs := make(chan error, 24)
			for i := 0; i < 24; i++ {
				wg.Add(1)
				go func(i int) { defer wg.Done(); _, e := s.Checkout(ctx, fmt.Sprint(i), "sku", 10); errs <- e }(i)
			}
			wg.Wait()
			close(errs)
			for e := range errs {
				if e != nil {
					return e
				}
			}
			return nil
		})
		if v := b.Snapshot(); e != nil || v.Charges != 24 || v.ReserveCalls != 24 {
			t.Fatalf("concurrent checkout: %v %+v", e, v)
		}
	})
	t.Run("canceled_before_reserve", func(t *testing.T) {
		b := &Backend{}
		s := service(t, b)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, e := s.Checkout(ctx, "u", "sku", 10)
		v := b.Snapshot()
		if !errors.Is(e, context.Canceled) || v.ReserveCalls != 0 || v.Charges != 0 {
			t.Fatalf("canceled checkout applied work: %v %+v", e, v)
		}
	})
}
