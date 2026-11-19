package t1

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk"
)

func newSvc(t *testing.T, b *Backend) *Service {
	s, e := New(b)
	if e != nil {
		t.Fatal(e)
	}
	return s
}

func TestRetriesUnavailable(t *testing.T) {
	b := &Backend{ChargeError: func(c int) error {
		if c < 3 {
			return fmt.Errorf("wrapped: %w", ErrUnavailable)
		}
		return nil
	}}
	r, e := newSvc(t, b).Charge(context.Background(), "a", 5)
	v := b.Snapshot()
	if e != nil || r.Cents != 5 || v.ChargeCalls != 3 || v.Charges != 1 {
		t.Fatalf("%+v %v %+v", r, e, v)
	}
}

func TestGivesUpAfterThree(t *testing.T) {
	b := &Backend{ChargeError: func(int) error { return ErrUnavailable }}
	_, e := newSvc(t, b).Charge(context.Background(), "a", 5)
	if !errors.Is(e, ErrUnavailable) || b.Snapshot().ChargeCalls != 3 || hertasdk.OutcomeOf(e) != hertasdk.Transient {
		t.Fatalf("%v %+v", e, b.Snapshot())
	}
}

func TestTimeoutNotRetried(t *testing.T) {
	for _, err := range []error{ErrTimeout, errors.Join(ErrUnavailable, ErrTimeout)} {
		b := &Backend{ChargeError: func(int) error { return err }}
		_, e := newSvc(t, b).Charge(context.Background(), "a", 5)
		v := b.Snapshot()
		if !errors.Is(e, ErrTimeout) || hertasdk.OutcomeOf(e) != hertasdk.Uncertain || v.ChargeCalls != 1 || v.Charges != 1 {
			t.Fatalf("%v %+v", e, v)
		}
	}
}

func TestDeclined(t *testing.T) {
	b := &Backend{ChargeError: func(int) error { return ErrDeclined }}
	_, e := newSvc(t, b).Charge(context.Background(), "a", 5)
	if !errors.Is(e, ErrDeclined) || b.Snapshot().ChargeCalls != 1 || hertasdk.OutcomeOf(e) != hertasdk.Permanent {
		t.Fatalf("%v %+v", e, b.Snapshot())
	}
}

func TestPerCallTimeout(t *testing.T) {
	b := &Backend{ChargeWait: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }}
	start := time.Now()
	_, e := newSvc(t, b).Charge(context.Background(), "a", 5)
	d := time.Since(start)
	if e == nil || d < 450*time.Millisecond || d > 900*time.Millisecond || b.Snapshot().ChargeCalls != 1 {
		t.Fatalf("%v %v %+v", e, d, b.Snapshot())
	}
	if hertasdk.OutcomeOf(e) != hertasdk.Uncertain {
		t.Fatalf("outcome %v", hertasdk.OutcomeOf(e))
	}
}

func TestCallerCancel(t *testing.T) {
	b := &Backend{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := newSvc(t, b).Charge(ctx, "a", 5)
	if !errors.Is(e, context.Canceled) || b.Snapshot().Charges != 0 {
		t.Fatal(e)
	}
}
