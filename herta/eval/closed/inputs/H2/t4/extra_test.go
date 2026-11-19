package t4

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestTransfersConcurrent(t *testing.T) {
	b := &Backend{}
	s, _ := New(b)
	ids := []string{"", "a", "b", "\x00transfer\x00"}
	for _, id := range ids {
		b.SetBalance(id, 1000)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < 400; i++ {
		f, to := ids[i%4], ids[(i/4)%4]
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := s.Transfer(ctx, f, to, 1); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	var sum int64
	for _, v := range b.Snapshot().Balances {
		sum += v
	}
	if sum != 4000 {
		t.Fatal(b.Snapshot().Balances)
	}
}
