package t1

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

func TestOracleHardCharge(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"wrapped", fmt.Errorf("gateway: %w", ErrTimeout)},
		{"joined", errors.Join(errors.New("connection reset"), ErrTimeout)},
		{"joined_unavailable", errors.Join(ErrUnavailable, ErrTimeout)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := &Backend{ChargeError: func(call int) error {
				if call == 1 {
					return tc.err
				}
				return nil
			}}
			s := service(t, b)
			_, e := s.Charge(context.Background(), "u", 10)
			v := b.Snapshot()
			if !errors.Is(e, ErrTimeout) || hertasdk.OutcomeOf(e) != hertasdk.Uncertain || v.Charges != 1 || v.ChargeCalls != 1 {
				t.Fatalf("uncertain charge repeated/misreported: err=%v outcome=%v charges=%d calls=%d", e, hertasdk.OutcomeOf(e), v.Charges, v.ChargeCalls)
			}
		})
	}
	t.Run("caller_deadline", func(t *testing.T) {
		b := &Backend{ChargeWait: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }}
		s := service(t, b)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()
		_, e := s.Charge(ctx, "u", 10)
		v := b.Snapshot()
		if !errors.Is(e, context.DeadlineExceeded) || v.ChargeCalls != 1 || v.Charges != 0 {
			t.Fatalf("caller deadline: %v %+v", e, v)
		}
	})
	t.Run("three_independent_requests", func(t *testing.T) {
		b := &Backend{}
		s := service(t, b)
		for i := 0; i < 3; i++ {
			if _, e := s.Charge(context.Background(), "u", 10); e != nil {
				t.Fatal(e)
			}
		}
		if v := b.Snapshot(); v.Charges != 3 {
			t.Fatalf("independent payments suppressed: %+v", v)
		}
	})
}
