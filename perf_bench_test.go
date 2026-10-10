package hertasdk

import (
	"context"
	"testing"
	"time"
)

// Uncontended benchmarks: one caller, so every acquisition takes the fast
// path. They are the A5 cost model: what a call pays when nothing waits.

func uncontendedOp(tb testing.TB, timeout time.Duration, key func(int) string) *Operation[int, int] {
	tb.Helper()
	rt, err := NewRuntime(ResourceSpec{Name: "worker", Capacity: 4})
	if err != nil {
		tb.Fatal(err)
	}
	op, err := NewOperation(rt, "work",
		func(_ context.Context, i int) (int, error) { return i, nil },
		Policy[int]{
			Effect:       Pure,
			Resources:    []Requirement{{Name: "worker", Units: 1}},
			Timeout:      timeout,
			SerializeKey: key,
		})
	if err != nil {
		tb.Fatal(err)
	}
	return op
}

func benchSerial(b *testing.B, op *Operation[int, int]) {
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		if _, err := op.Do(ctx, i); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDoUncontended(b *testing.B) {
	benchSerial(b, uncontendedOp(b, 0, nil))
}

func BenchmarkDoUncontendedTimeout(b *testing.B) {
	benchSerial(b, uncontendedOp(b, time.Hour, nil))
}

func BenchmarkDoUncontendedKeyed(b *testing.B) {
	benchSerial(b, uncontendedOp(b, 0, func(int) string { return "brand-1" }))
}
