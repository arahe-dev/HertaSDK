package t1

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk"
)

func TestChargeRetryPolicy(t *testing.T) {
	tests := []struct {
		name   string
		fails  []error
		cause  error
		result hertasdk.Outcome
		calls  int
		paid   int
	}{
		{"recovers", []error{fmt.Errorf("provider: %w", ErrUnavailable), ErrUnavailable}, nil, hertasdk.Success, 3, 1},
		{"exhausted", []error{ErrUnavailable, ErrUnavailable, ErrUnavailable}, ErrUnavailable, hertasdk.Transient, 3, 0},
		{"possibly applied", []error{ErrUnavailable, fmt.Errorf("provider: %w", ErrTimeout)}, ErrTimeout, hertasdk.Uncertain, 2, 1},
		{"declined", []error{ErrDeclined}, ErrDeclined, hertasdk.Permanent, 1, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &Backend{ChargeError: func(call int) error {
				if call <= len(tc.fails) {
					return tc.fails[call-1]
				}
				return nil
			}}
			s, err := New(b)
			if err != nil {
				t.Fatal(err)
			}
			r, err := s.Charge(context.Background(), "alice", 42)
			if !errors.Is(err, tc.cause) || hertasdk.OutcomeOf(err) != tc.result {
				t.Fatalf("error = %v (%s), want %v (%s)", err, hertasdk.OutcomeOf(err), tc.cause, tc.result)
			}
			if tc.cause == nil && (r.User != "alice" || r.Cents != 42 || r.ID != 1) {
				t.Fatalf("receipt = %+v", r)
			}
			if got := b.Snapshot(); got.ChargeCalls != tc.calls || got.Charges != tc.paid {
				t.Fatalf("calls = %d, charges = %d; want %d, %d", got.ChargeCalls, got.Charges, tc.calls, tc.paid)
			}
		})
	}
}

func TestChargeAttemptDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	b := &Backend{ChargeWait: func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if remaining := time.Until(deadline); !ok || remaining <= 0 || remaining > 500*time.Millisecond {
			return fmt.Errorf("expected a 500ms attempt deadline, got %s", remaining)
		}
		<-ctx.Done()
		return ctx.Err()
	}}
	s, err := New(b)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Charge(ctx, "alice", 42)
	if !errors.Is(err, context.DeadlineExceeded) || hertasdk.OutcomeOf(err) != hertasdk.Uncertain || ctx.Err() != nil {
		t.Fatalf("attempt error = %v (%s), caller error = %v", err, hertasdk.OutcomeOf(err), ctx.Err())
	}
	if got := b.Snapshot(); got.ChargeCalls != 1 || got.Charges != 0 {
		t.Fatalf("calls = %d, charges = %d; want 1, 0", got.ChargeCalls, got.Charges)
	}
}
