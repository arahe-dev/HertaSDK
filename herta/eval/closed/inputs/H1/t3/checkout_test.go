package t3

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCheckoutFailureBoundaries(t *testing.T) {
	tests := []struct {
		name         string
		reserve      []error
		charge       error
		want         error
		reserveCalls int
		chargeCalls  int
		charges      int
	}{
		{"reserve recovers", []error{ErrTimeout, ErrUnavailable}, nil, nil, 3, 1, 1},
		{"reserve exhausts retries", []error{ErrTimeout, ErrUnavailable, ErrTimeout}, nil, ErrTimeout, 3, 0, 0},
		{"reserve rejects", []error{ErrDeclined}, nil, ErrDeclined, 1, 0, 0},
		{"charge is uncertain", []error{ErrUnavailable}, ErrTimeout, ErrTimeout, 2, 1, 1},
		{"charge rejects", nil, ErrDeclined, ErrDeclined, 1, 1, 0},
		{"preserve charge retries", nil, ErrUnavailable, ErrUnavailable, 1, 3, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &Backend{
				ReserveError: func(call int) error {
					if call <= len(tc.reserve) {
						return tc.reserve[call-1]
					}
					return nil
				},
				ChargeError: func(int) error { return tc.charge },
			}
			s, err := New(b)
			if err != nil {
				t.Fatal(err)
			}
			r, err := s.Checkout(context.Background(), "alice", "book", 42)
			if !errors.Is(err, tc.want) {
				t.Fatalf("checkout error = %v, want %v", err, tc.want)
			}
			if tc.want == nil && (r.User != "alice" || r.Cents != 42 || r.ID != 1) {
				t.Fatalf("receipt = %+v", r)
			}
			got := b.Snapshot()
			if got.ReserveCalls != tc.reserveCalls || got.ChargeCalls != tc.chargeCalls || got.Charges != tc.charges {
				t.Fatalf("checkout snapshot: %+v", got)
			}
		})
	}
}

func TestConcurrentCheckoutMakesProgress(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	b := &Backend{}
	s, err := New(b)
	if err != nil {
		t.Fatal(err)
	}
	const requests = 24
	start := make(chan struct{})
	done := make(chan error, requests)
	for range requests {
		go func() {
			<-start
			_, err := s.Checkout(ctx, "alice", "book", 42)
			done <- err
		}()
	}
	close(start)
	for range requests {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if got := b.Snapshot(); got.ReserveCalls != requests || got.ChargeCalls != requests || got.Charges != requests {
		t.Fatalf("checkout snapshot: %+v", got)
	}
}
