package t4

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestConcurrentTransfers(t *testing.T) {
	b := &Backend{}
	s, _ := New(b)
	users := []string{"", "a", "transfer:", "user:", "transfer:user:", "user:transfer:", "c"}
	for _, u := range users {
		b.SetBalance(u, 1000)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < 20*len(users)*len(users); i++ {
		from, to := users[i%len(users)], users[(i/len(users))%len(users)]
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := s.Transfer(ctx, from, to, 1); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	var sum int64
	for _, v := range b.Snapshot().Balances {
		sum += v
	}
	if sum != int64(1000*len(users)) {
		t.Fatal(sum)
	}
}

func TestInsufficient(t *testing.T) {
	b := &Backend{}
	s, _ := New(b)
	b.SetBalance("a", 5)
	if e := s.Transfer(context.Background(), "a", "b", 10); !errors.Is(e, ErrDeclined) {
		t.Fatal(e)
	}
	if v := b.Snapshot(); v.CreditCalls != 0 || v.Balances["a"] != 5 {
		t.Fatal(v)
	}
}

func TestSelfTransfer(t *testing.T) {
	b := &Backend{}
	s, _ := New(b)
	b.SetBalance("", 5)
	if e := s.Transfer(context.Background(), "", "", 5); e != nil || b.Snapshot().Balances[""] != 5 {
		t.Fatal(e, b.Snapshot())
	}
}
