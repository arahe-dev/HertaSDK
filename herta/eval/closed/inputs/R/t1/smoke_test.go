package t1

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
	r, e := s.Charge(ctx, "alice", 10)
	if e != nil || r.Cents != 10 || b.Snapshot().Charges != 1 {
		t.Fatalf("charge: %+v %v", r, e)
	}
}
