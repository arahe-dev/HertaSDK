package t3

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
	r, e := s.Checkout(ctx, "alice", "book", 10)
	if e != nil || r.Cents != 10 {
		t.Fatalf("checkout: %+v %v", r, e)
	}
	if b.Snapshot().ReserveCalls != 1 || b.Snapshot().Charges != 1 {
		t.Fatal("checkout incomplete")
	}
}
