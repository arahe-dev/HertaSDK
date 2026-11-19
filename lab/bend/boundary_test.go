package bend_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arahe-dev/hertasdk"
)

func TestRetryEvidenceBoundary(t *testing.T) {
	for _, tc := range []struct {
		name        string
		firstEffect bool
		reported    hertasdk.Outcome
		wantCalls   int
		wantCharges int
		wantOutcome hertasdk.Outcome
	}{
		{"rejected before charge", false, hertasdk.Transient, 2, 1, hertasdk.Success},
		{"applied but ack lost", true, hertasdk.Uncertain, 1, 1, hertasdk.Uncertain},
		{"applied but ack lost and reported as rejected", true, hertasdk.Transient, 2, 2, hertasdk.Success},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt, err := hertasdk.NewRuntime(hertasdk.ResourceSpec{Name: "payments", Capacity: 1})
			if err != nil {
				t.Fatal(err)
			}
			lostAck := errors.New("first attempt failed")
			var calls, charges int
			charge, err := hertasdk.NewOperation(rt, "charge", func(context.Context, struct{}) (struct{}, error) {
				calls++
				if calls == 1 {
					if tc.firstEffect {
						charges++
					}
					return struct{}{}, hertasdk.Fail(tc.reported, lostAck)
				}
				charges++
				return struct{}{}, nil
			}, hertasdk.Policy[struct{}]{
				Effect:    hertasdk.NonIdempotent,
				Resources: []hertasdk.Requirement{{Name: "payments", Units: 1}},
				Retry:     hertasdk.RetryPolicy{MaxAttempts: 2, On: map[hertasdk.Outcome]bool{hertasdk.Transient: true}},
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = charge.Do(context.Background(), struct{}{})
			if got := hertasdk.OutcomeOf(err); got != tc.wantOutcome || calls != tc.wantCalls || charges != tc.wantCharges {
				t.Fatalf("got %s (err: %v), %d calls and %d charges; want %s, %d calls and %d charges",
					got, err, calls, charges, tc.wantOutcome, tc.wantCalls, tc.wantCharges)
			}
			t.Logf("%s, %d calls, %d charges", tc.wantOutcome, calls, charges)
		})
	}
}
