package t4

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
	b.SetBalance("", 100)
	if e := s.Transfer(ctx, "", "bob", 10); e != nil {
		t.Fatal(e)
	}
	v := b.Snapshot()
	if v.Balances[""] != 90 || v.Balances["bob"] != 10 {
		t.Fatalf("balances: %v", v.Balances)
	}
}
