package t2

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

func TestOracleRefund(t *testing.T) {
	t.Run("same_order", func(t *testing.T) {
		b := &Backend{RefundDelay: 5 * time.Millisecond}
		s := service(t, b)
		e := bounded(t, func(ctx context.Context) error {
			var wg sync.WaitGroup
			errs := make(chan error, 24)
			for i := 0; i < 24; i++ {
				wg.Add(1)
				go func() { defer wg.Done(); errs <- s.Refund(ctx, -9223372036854775808, 1) }()
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
		v := b.Snapshot()
		if e != nil || v.RefundOverlap || v.Refunds != 24 {
			t.Fatalf("same-order serialization: %v %+v", e, v)
		}
	})
	t.Run("different_orders", func(t *testing.T) {
		entered := make(chan int64, 2)
		release := make(chan struct{})
		b := &Backend{RefundEntered: func(id int64) { entered <- id }, RefundRelease: release}
		s := service(t, b)
		e := bounded(t, func(ctx context.Context) error {
			done := make(chan error, 2)
			go func() { done <- s.Refund(ctx, 0, 0) }()
			go func() { done <- s.Refund(ctx, 9223372036854775807, 10) }()
			timer := time.NewTimer(500 * time.Millisecond)
			defer timer.Stop()
			for i := 0; i < 2; i++ {
				select {
				case <-entered:
				case <-timer.C:
					close(release)
					return errors.New("different orders did not enter backend concurrently")
				}
			}
			close(release)
			e1, e2 := <-done, <-done
			if e1 != nil {
				return e1
			}
			return e2
		})
		if e != nil || b.Snapshot().MaxRefundActive < 2 {
			t.Fatalf("parallel orders: %v %+v", e, b.Snapshot())
		}
	})
	t.Run("error", func(t *testing.T) {
		b := &Backend{RefundError: func(int64, int) error { return ErrDeclined }}
		s := service(t, b)
		if e := s.Refund(context.Background(), 0, 0); !errors.Is(e, ErrDeclined) {
			t.Fatalf("refund error lost: %v", e)
		}
	})
}
