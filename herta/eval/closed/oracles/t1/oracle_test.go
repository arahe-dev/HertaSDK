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

func TestOracleCharge(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		b := &Backend{}
		s := service(t, b)
		r, e := s.Charge(context.Background(), "", 0)
		if e != nil || r.User != "" || r.Cents != 0 || b.Snapshot().Charges != 1 {
			t.Fatalf("zero-valued payment: %+v %v %+v", r, e, b.Snapshot())
		}
	})
	t.Run("timeout", func(t *testing.T) {
		b := &Backend{ChargeError: func(call int) error {
			if call == 1 {
				return fmt.Errorf("provider: %w", ErrTimeout)
			}
			return nil
		}}
		s := service(t, b)
		_, e := s.Charge(context.Background(), "alice", 70)
		v := b.Snapshot()
		if !errors.Is(e, ErrTimeout) || hertasdk.OutcomeOf(e) != hertasdk.Uncertain || v.Charges != 1 || v.ChargeCalls != 1 {
			t.Fatalf("ambiguous payment repeated or misreported: err=%v outcome=%v charges=%d calls=%d", e, hertasdk.OutcomeOf(e), v.Charges, v.ChargeCalls)
		}
	})
	t.Run("unavailable", func(t *testing.T) {
		b := &Backend{ChargeError: func(call int) error {
			if call < 3 {
				return fmt.Errorf("provider: %w", ErrUnavailable)
			}
			return nil
		}}
		s := service(t, b)
		_, e := s.Charge(context.Background(), "a", 7)
		v := b.Snapshot()
		if e != nil || v.ChargeCalls != 3 || v.Charges != 1 {
			t.Fatalf("safe retry: err=%v snapshot=%+v", e, v)
		}
	})
	t.Run("bounded", func(t *testing.T) {
		b := &Backend{ChargeError: func(int) error { return ErrUnavailable }}
		s := service(t, b)
		_, e := s.Charge(context.Background(), "a", 7)
		v := b.Snapshot()
		if !errors.Is(e, ErrUnavailable) || hertasdk.OutcomeOf(e) != hertasdk.Transient || v.ChargeCalls != 3 || v.Charges != 0 {
			t.Fatalf("attempt limit: err=%v snapshot=%+v", e, v)
		}
	})
	t.Run("declined", func(t *testing.T) {
		b := &Backend{ChargeError: func(int) error { return ErrDeclined }}
		s := service(t, b)
		_, e := s.Charge(context.Background(), "a", 7)
		if !errors.Is(e, ErrDeclined) || hertasdk.OutcomeOf(e) != hertasdk.Permanent || b.Snapshot().ChargeCalls != 1 {
			t.Fatalf("decline: %v %+v", e, b.Snapshot())
		}
	})
	t.Run("deadline", func(t *testing.T) {
		b := &Backend{ChargeWait: func(ctx context.Context) error {
			deadline, ok := ctx.Deadline()
			if !ok {
				return errors.New("missing operation deadline")
			}
			if remaining := time.Until(deadline); remaining > 510*time.Millisecond || remaining < 400*time.Millisecond {
				return fmt.Errorf("unexpected deadline: %v", remaining)
			}
			<-ctx.Done()
			return ctx.Err()
		}}
		s := service(t, b)
		start := time.Now()
		e := bounded(t, func(ctx context.Context) error { _, e := s.Charge(ctx, "a", 7); return e })
		if !errors.Is(e, context.DeadlineExceeded) || time.Since(start) > time.Second || b.Snapshot().ChargeCalls != 1 {
			t.Fatalf("500ms deadline not honored: %v elapsed=%v %+v", e, time.Since(start), b.Snapshot())
		}
	})
	t.Run("canceled", func(t *testing.T) {
		b := &Backend{}
		s := service(t, b)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, e := s.Charge(ctx, "a", 7)
		if !errors.Is(e, context.Canceled) || b.Snapshot().Charges != 0 {
			t.Fatalf("cancellation: %v %+v", e, b.Snapshot())
		}
	})
}
