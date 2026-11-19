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

var _ = fmt.Sprint
var _ sync.WaitGroup
var _ = time.Second
var _ = hertasdk.OutcomeOf

func TestOracleHardRefund(t *testing.T) {
	t.Run("key_waiter_cancellation", func(t *testing.T) {
		entered := make(chan int64, 8)
		release := make(chan struct{})
		defer close(release)
		b := &Backend{RefundEntered: func(id int64) { entered <- id }, RefundRelease: release}
		s := service(t, b)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		done := make(chan error, 2)
		go func() { done <- s.Refund(ctx, -1, 10) }()
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal("first refund did not enter")
		}
		waitCtx, stop := context.WithTimeout(ctx, 30*time.Millisecond)
		defer stop()
		if e := s.Refund(waitCtx, -1, 10); !errors.Is(e, context.DeadlineExceeded) {
			t.Fatalf("blocked same-order refund: %v", e)
		}
		if v := b.Snapshot(); v.Refunds != 1 || v.RefundOverlap {
			t.Fatalf("canceled waiter entered backend: %+v", v)
		}
		go func() { done <- s.Refund(ctx, 1, 10) }()
		select {
		case id := <-entered:
			if id != 1 {
				t.Fatalf("unexpected refund %d", id)
			}
		case <-ctx.Done():
			t.Fatal("independent order blocked")
		}
		cancel()
		for i := 0; i < 2; i++ {
			<-done
		}
	})
	t.Run("many_distinct_signed_orders", func(t *testing.T) {
		b := &Backend{RefundDelay: time.Millisecond}
		s := service(t, b)
		e := bounded(t, func(ctx context.Context) error {
			var wg sync.WaitGroup
			errs := make(chan error, 48)
			for i := 0; i < 48; i++ {
				wg.Add(1)
				go func(i int) { defer wg.Done(); errs <- s.Refund(ctx, int64(i%6-3), uint64(i)) }(i)
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
		if v := b.Snapshot(); e != nil || v.Refunds != 48 || v.RefundOverlap || v.MaxRefundActive < 2 {
			t.Fatalf("mixed orders: %v %+v", e, v)
		}
	})
}
