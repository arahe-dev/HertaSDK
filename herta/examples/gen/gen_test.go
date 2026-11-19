package gen

import (
	"context"
	"errors"
	"testing"

	"github.com/arahe-dev/hertasdk"
)

// The compiled charge op runs against hertasdk end to end: construction,
// admission, retry on the classified transient failure, and shutdown.
func TestChargeRunsAgainstSDK(t *testing.T) {
	rt, err := NewMainRuntime()
	if err != nil {
		t.Fatal(err)
	}
	charge, err := NewMainCharge(rt)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	got, err := charge.Do(ctx, MainChargeInput{UserId: "u1", Cents: 250})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if got.cents != 250 {
		t.Fatalf("receipt.cents = %d, want 250", got.cents)
	}

	// cents=0 makes do_charge report its provider error; the handler
	// classifies it transient, so the op retries max 3 times then returns
	// the Failure.
	_, err = charge.Do(ctx, MainChargeInput{UserId: "u2", Cents: 0})
	if err == nil {
		t.Fatal("expected failure")
	}
	if o := hertasdk.OutcomeOf(err); o != hertasdk.Transient {
		t.Fatalf("OutcomeOf(err) = %s, want Transient", o)
	}
	var f *hertasdk.Failure
	if !errors.As(err, &f) || !errors.Is(f.Err, errProvider) {
		t.Fatalf("error chain lost the provider error: %v", err)
	}

	stats := rt.Stats()
	if stats.Retried != 2 {
		t.Fatalf("Retried = %d, want 2 (retry max 3 on transient)", stats.Retried)
	}
	if err := rt.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}
