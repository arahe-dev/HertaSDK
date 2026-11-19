package t4

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

func TestOracleTransfer(t *testing.T) {
	t.Run("completes", func(t *testing.T) {
		b := &Backend{}
		b.SetBalance("alice", 100)
		s := service(t, b)
		e := bounded(t, func(ctx context.Context) error { return s.Transfer(ctx, "alice", "bob", 10) })
		v := b.Snapshot()
		if e != nil || v.Balances["alice"] != 90 || v.Balances["bob"] != 10 || v.DebitCalls != 1 || v.CreditCalls != 1 {
			t.Fatalf("transfer did not complete correctly: %v %+v", e, v)
		}
	})
	t.Run("concurrent", func(t *testing.T) {
		b := &Backend{}
		b.SetBalance("alice", 1000)
		s := service(t, b)
		e := bounded(t, func(ctx context.Context) error {
			var wg sync.WaitGroup
			errs := make(chan error, 20)
			for i := 0; i < 20; i++ {
				wg.Add(1)
				go func(i int) { defer wg.Done(); errs <- s.Transfer(ctx, "alice", fmt.Sprintf("recipient-%d", i), 10) }(i)
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
		if e != nil || v.Balances["alice"] != 800 || v.DebitCalls != 20 || v.CreditCalls != 20 {
			t.Fatalf("concurrent transfers: %v %+v", e, v)
		}
		for i := 0; i < 20; i++ {
			if v.Balances[fmt.Sprintf("recipient-%d", i)] != 10 {
				t.Fatalf("missing recipient %d: %+v", i, v)
			}
		}
	})
	t.Run("empty_account", func(t *testing.T) {
		b := &Backend{}
		b.SetBalance("", 10)
		s := service(t, b)
		e := bounded(t, func(ctx context.Context) error { return s.Transfer(ctx, "", "b", 0) })
		if e != nil || b.Snapshot().Balances[""] != 10 {
			t.Fatalf("empty account or zero transfer: %v %+v", e, b.Snapshot())
		}
	})
	t.Run("declined", func(t *testing.T) {
		b := &Backend{}
		s := service(t, b)
		e := bounded(t, func(ctx context.Context) error { return s.Transfer(ctx, "alice", "bob", 1) })
		v := b.Snapshot()
		if !errors.Is(e, ErrDeclined) || v.CreditCalls != 0 {
			t.Fatalf("debit rejection: %v %+v", e, v)
		}
	})
}
