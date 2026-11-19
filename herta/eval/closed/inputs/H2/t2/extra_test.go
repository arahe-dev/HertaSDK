package t2

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"
)

func TestRefundSerialization(t *testing.T) {
	b := &Backend{RefundDelay: 20 * time.Millisecond}
	s, _ := New(b)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		id := []int64{0, math.MinInt64, math.MaxInt64, -1}[i%4]
		wg.Add(1)
		go func() { defer wg.Done(); _ = s.Refund(context.Background(), id, 1) }()
	}
	wg.Wait()
	v := b.Snapshot()
	if v.RefundOverlap || v.Refunds != 20 || v.MaxRefundActive < 2 {
		t.Fatal(v)
	}
}
