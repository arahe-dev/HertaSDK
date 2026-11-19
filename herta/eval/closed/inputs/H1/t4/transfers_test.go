package t4

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestConcurrentTransfers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	b := &Backend{}
	initial := map[string]int64{"": 200, "transfer:": 200, "account:": 200}
	for account, balance := range initial {
		b.SetBalance(account, balance)
	}
	s, err := New(b)
	if err != nil {
		t.Fatal(err)
	}
	pairs := []struct{ from, to string }{
		{"", "transfer:"},
		{"transfer:", ""},
		{"", ""},
		{"", "account:"},
		{"account:", "transfer:"},
	}
	const repeats = 24
	start := make(chan struct{})
	done := make(chan error, repeats*len(pairs))
	want := initial
	for range repeats {
		for _, pair := range pairs {
			want[pair.from]--
			want[pair.to]++
			go func() {
				<-start
				done <- s.Transfer(ctx, pair.from, pair.to, 1)
			}()
		}
	}
	close(start)
	for range cap(done) {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatalf("transfers did not complete: %v", ctx.Err())
		}
	}
	got := b.Snapshot()
	for account, balance := range want {
		if got.Balances[account] != balance {
			t.Errorf("balance for %q = %d, want %d", account, got.Balances[account], balance)
		}
	}
	if got.DebitCalls != cap(done) || got.CreditCalls != cap(done) {
		t.Fatalf("transfers did not each debit and credit exactly once: %+v", got)
	}
}

func TestTransferStopsAfterFailedDebit(t *testing.T) {
	b := &Backend{}
	s, err := New(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Transfer(context.Background(), "", "bob", 1); !errors.Is(err, ErrDeclined) {
		t.Fatalf("transfer error = %v, want %v", err, ErrDeclined)
	}
	if got := b.Snapshot(); got.CreditCalls != 0 || got.Balances["bob"] != 0 {
		t.Fatalf("credited after failed debit: %+v", got)
	}
}
