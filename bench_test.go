package execution

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func BenchmarkContention(b *testing.B) {
	rt, err := NewRuntime(ResourceSpec{Name: "worker", Capacity: 4})
	if err != nil {
		b.Fatal(err)
	}
	var counter atomic.Uint64
	op, err := NewOperation(rt, "work",
		func(_ context.Context, _ int) (int, error) {
			counter.Add(1)
			return 0, nil
		},
		Policy[int]{Effect: Pure,
			Resources: []Requirement{{Name: "worker", Units: 1}},
			Admission: Wait,
		})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if _, err := op.Do(context.Background(), i); err != nil {
				b.Error(err)
				return
			}
			i++
		}
	})
}

func BenchmarkKeyedSerialization(b *testing.B) {
	rt, err := NewRuntime(ResourceSpec{Name: "worker", Capacity: 4})
	if err != nil {
		b.Fatal(err)
	}
	var counter atomic.Uint64
	op, err := NewOperation(rt, "keyed",
		func(_ context.Context, _ int) (int, error) {
			counter.Add(1)
			return 0, nil
		},
		Policy[int]{Effect: Pure,
			Resources: []Requirement{{Name: "worker", Units: 1}},
			Admission: Wait,
			SerializeKey: func(i int) string {
				switch i & 7 {
				case 0:
					return "key-0"
				case 1:
					return "key-1"
				case 2:
					return "key-2"
				case 3:
					return "key-3"
				case 4:
					return "key-4"
				case 5:
					return "key-5"
				case 6:
					return "key-6"
				default:
					return "key-7"
				}
			},
		})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if _, err := op.Do(context.Background(), i); err != nil {
				b.Error(err)
				return
			}
			i++
		}
	})
}

func BenchmarkMultiResource(b *testing.B) {
	rt, err := NewRuntime(
		ResourceSpec{Name: "cpu", Capacity: 4},
		ResourceSpec{Name: "io", Capacity: 4},
	)
	if err != nil {
		b.Fatal(err)
	}
	var counter atomic.Uint64
	op, err := NewOperation(rt, "multi",
		func(_ context.Context, _ int) (int, error) {
			counter.Add(1)
			return 0, nil
		},
		Policy[int]{Effect: Pure,
			Resources: []Requirement{
				{Name: "cpu", Units: 1},
				{Name: "io", Units: 1},
			},
			Admission: Wait,
		})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if _, err := op.Do(context.Background(), i); err != nil {
				b.Error(err)
				return
			}
			i++
		}
	})
}

func BenchmarkRetryOverhead(b *testing.B) {
	boom := errors.New("boom")
	mkOp := func(b *testing.B, maxAttempts int) *Operation[int, int] {
		b.Helper()
		rt, err := NewRuntime(ResourceSpec{Name: "worker", Capacity: 4})
		if err != nil {
			b.Fatal(err)
		}
		var counter atomic.Uint64
		op, err := NewOperation(rt, "flaky",
			func(_ context.Context, _ int) (int, error) {
				counter.Add(1)
				return 0, Fail(Transient, boom)
			},
			Policy[int]{
				Effect:    Idempotent,
				Resources: []Requirement{{Name: "worker", Units: 1}},
				Admission: Wait,
				Retry:     RetryPolicy{MaxAttempts: maxAttempts, On: map[Outcome]bool{Transient: true}},
			})
		if err != nil {
			b.Fatal(err)
		}
		return op
	}

	b.Run("retry/max_attempts=1", func(b *testing.B) {
		op := mkOp(b, 1)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_, _ = op.Do(context.Background(), i)
				i++
			}
		})
	})

	b.Run("retry/max_attempts=3", func(b *testing.B) {
		op := mkOp(b, 3)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_, _ = op.Do(context.Background(), i)
				i++
			}
		})
	})
}
