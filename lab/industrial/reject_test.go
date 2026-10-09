package industrial_test

import (
	"context"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// rejectShape is how callers react to a Reject refusal.
type rejectShape struct {
	name   string
	judged bool // recon must keep up with v0.2 under this shape
	run    func(op impl.Op, callers int, window time.Duration) (ok uint64)
}

var rejectShapes = []rejectShape{
	// A server: one goroutine per request, which exits after a refusal
	// (the refusal becomes a 429 for a remote client).
	{"server", true, func(op impl.Op, callers int, window time.Duration) uint64 {
		var ok atomic.Uint64
		inflight := make(chan struct{}, callers)
		for end := time.Now().Add(window); time.Now().Before(end); {
			inflight <- struct{}{}
			go func() {
				if _, err := op.Do(context.Background(), 0); err == nil {
					ok.Add(1)
				}
				<-inflight
			}()
		}
		for range callers {
			inflight <- struct{}{}
		}
		return ok.Load()
	}},
	// In-process callers that retry, yielding the processor after a refusal.
	{"yield", true, func(op impl.Op, callers int, window time.Duration) uint64 {
		return closedLoop(op, callers, window, runtime.Gosched)
	}},
	// In-process callers that retry at once. Without the refusal yield
	// (CONTRACT §4.1) they starve preempted holders once they outnumber
	// GOMAXPROCS (lab/results/r1/reject-probe/ROOT-CAUSE.md).
	{"spin", true, func(op impl.Op, callers int, window time.Duration) uint64 {
		return closedLoop(op, callers, window, func() {})
	}},
}

func closedLoop(op impl.Op, callers int, window time.Duration, onRefused func()) uint64 {
	var ok atomic.Uint64
	var stop atomic.Bool
	var wg sync.WaitGroup
	for range callers {
		wg.Go(func() {
			for !stop.Load() {
				if _, err := op.Do(context.Background(), 0); err == nil {
					ok.Add(1)
				} else {
					onRefused()
				}
			}
		})
	}
	time.Sleep(window)
	stop.Store(true)
	wg.Wait()
	return ok.Load()
}

// TestRejectUnderLoad drives a Reject budget of 4 far past capacity and
// measures the calls that get through. Implementations alternate within
// each replicate, so machine drift lands on both equally. Under every
// caller shape recon must deliver at least 0.9× v0.2.
func TestRejectUnderLoad(t *testing.T) {
	callers := workload.Scale([]int{64, 256}, []int{16, 64, 256, 1024})
	window := workload.Scale(300*time.Millisecond, time.Second)
	reps := workload.Scale(3, 5)
	impls := workload.Impls(t)

	for _, shape := range rejectShapes {
		for _, n := range callers {
			perSec := map[string][]float64{}
			for range reps {
				for _, im := range impls {
					op := rejectOp(t, im)
					perSec[im.Name()] = append(perSec[im.Name()], float64(shape.run(op, n, window))/window.Seconds())
					runtime.GC()
				}
			}
			name := fmt.Sprintf("%s/%d", shape.name, n)
			t.Run(name, func(t *testing.T) {
				med := map[string]float64{}
				for _, im := range impls {
					med[im.Name()] = median(perSec[im.Name()])
					t.Run(im.Name(), func(t *testing.T) {
						record(t, im, metrics{"ok_per_s": med[im.Name()], "callers": float64(n)})
					})
				}
				base, ok := med[impl.Orig.Name()]
				if !ok || !shape.judged {
					return
				}
				for _, im := range impls {
					if im == impl.Orig {
						continue
					}
					if ratio := med[im.Name()] / base; ratio < 0.9 {
						t.Errorf("%s: %s gets %.0f calls/s through, %.2f× v0.2 (%.0f); want ≥ 0.9×",
							name, im.Name(), med[im.Name()], ratio, base)
					}
				}
			})
		}
	}
}

func rejectOp(t *testing.T, im impl.Impl) impl.Op {
	t.Helper()
	rt, err := im.NewRuntime(impl.Spec{Name: "r", Capacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	op, err := rt.NewOp("op", impl.Policy{
		Effect:    impl.Idempotent,
		Admission: impl.Reject,
		Resources: []impl.Req{{Name: "r", Units: 1}},
	}, func(_ context.Context, in int) (int, error) { return in, nil })
	if err != nil {
		t.Fatal(err)
	}
	return op
}

func median(xs []float64) float64 {
	s := slices.Clone(xs)
	slices.Sort(s)
	return s[len(s)/2]
}
