package t2

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"
)

func TestAllIDs(t *testing.T) {
	b := &Backend{}
	s, _ := New(b)
	for _, id := range []int64{0, -1, math.MinInt64, math.MaxInt64} {
		if e := s.Refund(context.Background(), id, 1); e != nil {
			t.Fatal(e)
		}
	}
}

func TestSameOrderSerializedZero(t *testing.T) {
	for _, id := range []int64{0, -7, 42} {
		b := &Backend{RefundDelay: 5 * time.Millisecond}
		s, _ := New(b)
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); _ = s.Refund(context.Background(), id, 1) }()
		}
		wg.Wait()
		v := b.Snapshot()
		if v.RefundOverlap || v.Refunds != 10 {
			t.Fatalf("id %d: %+v", id, v)
		}
	}
}

func TestDifferentOrdersParallel(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan int64, 16)
	b := &Backend{RefundRelease: release, RefundEntered: func(id int64) { entered <- id }}
	s, _ := New(b)
	var wg sync.WaitGroup
	// Many same-order waiters must not starve other orders.
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = s.Refund(context.Background(), 1, 1) }()
	}
	for _, id := range []int64{2, 3, 4} {
		wg.Add(1)
		go func() { defer wg.Done(); _ = s.Refund(context.Background(), id, 1) }()
	}
	timeout := time.After(2 * time.Second)
	seen := map[int64]bool{}
	for len(seen) < 4 {
		select {
		case id := <-entered:
			seen[id] = true
		case <-timeout:
			t.Fatalf("not parallel: %v", seen)
		}
	}
	close(release)
	wg.Wait()
	if v := b.Snapshot(); v.RefundOverlap || v.MaxRefundActive != 4 {
		t.Fatalf("%+v", v)
	}
}

func TestErrorPropagates(t *testing.T) {
	bad := errors.New("boom")
	b := &Backend{RefundError: func(int64, int) error { return bad }}
	s, _ := New(b)
	if e := s.Refund(context.Background(), 1, 1); !errors.Is(e, bad) {
		t.Fatal(e)
	}
}
