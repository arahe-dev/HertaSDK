package t2

import (
	"context"
	"testing"
	"time"
)

func TestSmoke(t *testing.T) {
	b := &Backend{}
	s, e := New(b)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if e := s.Refund(ctx, 42, 10); e != nil {
		t.Fatal(e)
	}
	if b.Snapshot().Refunds != 1 {
		t.Fatal("refund missing")
	}
}
