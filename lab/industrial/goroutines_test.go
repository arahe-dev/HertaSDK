package industrial_test

import (
	"context"
	rtmetrics "runtime/metrics"
	"sync"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestGoroutinesPerDo counts, with the scheduler's own counter, the
// goroutines created per Do beyond the callers themselves. Δ10: none, on
// every path. The contended path parks 64 callers on a budget of 4.
func TestGoroutinesPerDo(t *testing.T) {
	n := workload.Scale(20_000, 200_000)
	paths := []struct {
		name    string
		callers int
		timeout time.Duration
		cancel  bool // each call gets its own cancellable context
	}{
		{"uncontended", 1, 0, false},
		{"uncontended_timeout", 1, time.Hour, false},
		{"contended", 64, 0, false},
		{"contended_cancellable", 64, 0, true},
	}
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		m := metrics{}
		for _, p := range paths {
			rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 4})
			if err != nil {
				t.Fatal(err)
			}
			pol := impl.Policy{Effect: impl.Pure, Resources: []impl.Req{{Name: "r", Units: 1}}, Timeout: p.timeout}
			op, err := rt.NewOp("op", pol, func(_ context.Context, in int) (int, error) { return in, nil })
			if err != nil {
				t.Fatal(err)
			}
			before := created()
			var wg sync.WaitGroup
			for c := range p.callers {
				wg.Go(func() {
					for i := c; i < n; i += p.callers {
						ctx, cancel := context.Background(), context.CancelFunc(func() {})
						if p.cancel {
							ctx, cancel = context.WithCancel(ctx)
						}
						op.Do(ctx, i)
						cancel()
					}
				})
			}
			wg.Wait()
			per := float64(created()-before-uint64(p.callers)) / float64(n)
			m[p.name+"_goroutines_per_do"] = per
			budget(t, im, workload.D10, p.name+" goroutines per Do", per, 0.001)
			rt.Shutdown(context.Background())
		}
		record(t, im, m)
	})
}

// created is the number of goroutines the process has ever created.
func created() uint64 {
	s := []rtmetrics.Sample{{Name: "/sched/goroutines-created:goroutines"}}
	rtmetrics.Read(s)
	return s[0].Value.Uint64()
}
