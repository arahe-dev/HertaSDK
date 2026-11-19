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

var _ = fmt.Sprint
var _ sync.WaitGroup
var _ = time.Second
var _ = hertasdk.OutcomeOf

func hardTransferBound(t *testing.T, f func(context.Context) error) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- f(ctx) }()
	select {
	case e := <-done:
		return e
	case <-time.After(700 * time.Millisecond):
		t.Fatal("transfer exceeded hard watchdog")
		return nil
	}
}
func TestOracleHardTransfer(t *testing.T) {
	t.Run("opposite_directions", func(t *testing.T) {
		b := &Backend{}
		b.SetBalance("A", 1000)
		b.SetBalance("B", 1000)
		s := service(t, b)
		e := hardTransferBound(t, func(ctx context.Context) error {
			var wg sync.WaitGroup
			errs := make(chan error, 40)
			start := make(chan struct{})
			for i := 0; i < 40; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					<-start
					from, to := "A", "B"
					if i%2 == 1 {
						from, to = to, from
					}
					errs <- s.Transfer(ctx, from, to, 1)
				}(i)
			}
			close(start)
			wg.Wait()
			close(errs)
			for e := range errs {
				if e != nil {
					return e
				}
			}
			return nil
		})
		if v := b.Snapshot(); e != nil || v.Balances["A"] != 1000 || v.Balances["B"] != 1000 || v.DebitCalls != 40 || v.CreditCalls != 40 {
			t.Fatalf("opposite transfers: %v %+v", e, v)
		}
	})
	// These are ordinary account IDs, including observed lock-prefix spellings.
	for _, pair := range [][2]string{{"", "B"}, {"A", ""}, {"A", "transfer:A"}, {"A", "account:A"}, {"A", "user:A"}, {"A", "\x00transfer\x00A"}, {"transfer:A", "A"}, {"user:", "transfer:"}, {"account:", "user:"}} {
		t.Run(fmt.Sprintf("accounts_%q_%q", pair[0], pair[1]), func(t *testing.T) {
			b := &Backend{}
			b.SetBalance(pair[0], 100)
			s := service(t, b)
			e := hardTransferBound(t, func(ctx context.Context) error { return s.Transfer(ctx, pair[0], pair[1], 10) })
			v := b.Snapshot()
			if e != nil || v.Balances[pair[0]] != 90 || v.Balances[pair[1]] != 10 || v.DebitCalls != 1 || v.CreditCalls != 1 {
				t.Fatalf("arbitrary account transfer: %v %+v", e, v)
			}
		})
	}
	t.Run("canceled", func(t *testing.T) {
		b := &Backend{}
		b.SetBalance("A", 100)
		s := service(t, b)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		e := s.Transfer(ctx, "A", "B", 10)
		v := b.Snapshot()
		if !errors.Is(e, context.Canceled) || v.Balances["A"] != 100 || v.DebitCalls != 0 || v.CreditCalls != 0 {
			t.Fatalf("canceled transfer: %v %+v", e, v)
		}
	})
}
