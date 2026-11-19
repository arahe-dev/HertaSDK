package t2

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

func TestRefundAdmissionByOrder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	entered := make(chan int64, 6)
	release := make(chan struct{})
	b := &Backend{
		RefundEntered: func(id int64) { entered <- id },
		RefundRelease: release,
	}
	s, err := New(b)
	if err != nil {
		t.Fatal(err)
	}
	orders := []int64{math.MinInt64, 0, math.MaxInt64}
	done := make(chan error, len(orders))
	for _, id := range orders {
		go func() { done <- s.Refund(ctx, id, 42) }()
	}
	active := make(map[int64]bool)
	for range orders {
		select {
		case id := <-entered:
			active[id] = true
		case <-ctx.Done():
			t.Fatalf("different orders did not all enter concurrently: %v", active)
		}
	}
	for _, id := range orders {
		if !active[id] {
			t.Fatalf("order %d not seen by backend", id)
		}
		waitCtx, stop := context.WithTimeout(ctx, 20*time.Millisecond)
		err := s.Refund(waitCtx, id, 21)
		stop()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("same order %d should wait; got %v", id, err)
		}
	}
	close(release)
	for range orders {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if got := b.Snapshot(); got.RefundOverlap || got.MaxRefundActive != len(orders) || got.Refunds != len(orders) {
		t.Fatalf("refund snapshot: %+v", got)
	}
}

func TestRefundPropagatesProviderError(t *testing.T) {
	b := &Backend{RefundError: func(int64, int) error { return ErrUnavailable }}
	s, err := New(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Refund(context.Background(), -1, 42); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("refund error = %v, want %v", err, ErrUnavailable)
	}
	if got := b.Snapshot().Refunds; got != 1 {
		t.Fatalf("refund attempts = %d, want 1", got)
	}
}
