package doe

import (
	"fmt"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// Workload is everything one run does to an implementation. The zero value
// is not runnable; start from Baseline.
type Workload struct {
	// Load shape. Rate 0 is a closed loop of Clients callers; Rate > 0 is an
	// open loop of Poisson arrivals served by up to Clients in flight, with
	// latency measured from the scheduled arrival (no coordinated omission).
	Clients int
	Rate    float64

	Warmup, Duration time.Duration

	// Budgets: Resources budgets r0.., each of Capacity units. Each call
	// claims 1 unit of every budget, or HeavyUnits with prob. HeavyFrac.
	Capacity   int64
	Resources  int
	HeavyFrac  float64
	HeavyUnits int64 // 0 means Capacity/2

	// Handler: busy-spins Service (exponentially distributed when ExpService)
	// and returns Fail(Transient) with probability FailRate per attempt.
	Service    time.Duration
	ExpService bool
	FailRate   float64

	// Policy. MaxAttempts > 1 retries Transient and Uncertain (Idempotent).
	Admission   impl.Admission
	Timeout     time.Duration
	MaxAttempts int

	// Keys > 0 serializes calls on one of Keys keys, Zipf(ZipfS) skewed when
	// ZipfS > 1, else uniform.
	Keys  int
	ZipfS float64

	// CancelRate is the fraction of calls whose context is cancelled at a
	// uniform random time in [0, 2·Service + 50µs).
	CancelRate float64

	Seed uint64
}

// Baseline is the reference workload every design perturbs.
func Baseline() Workload {
	return Workload{
		Clients:     8,
		Warmup:      100 * time.Millisecond,
		Duration:    300 * time.Millisecond,
		Capacity:    4,
		Resources:   1,
		MaxAttempts: 1,
		Seed:        1,
	}
}

func (w Workload) heavyUnits() int64 {
	if w.HeavyUnits > 0 {
		return w.HeavyUnits
	}
	return max(w.Capacity/2, 1)
}

func (w Workload) validate() error {
	switch {
	case w.Clients < 1:
		return fmt.Errorf("doe: Clients %d < 1", w.Clients)
	case w.Capacity < 1 || w.Resources < 1:
		return fmt.Errorf("doe: need Capacity ≥ 1 and Resources ≥ 1")
	case w.heavyUnits() > w.Capacity:
		return fmt.Errorf("doe: HeavyUnits %d > Capacity %d", w.heavyUnits(), w.Capacity)
	case w.Duration <= 0:
		return fmt.Errorf("doe: Duration must be > 0")
	case w.ZipfS != 0 && w.ZipfS <= 1:
		return fmt.Errorf("doe: ZipfS must be 0 (uniform) or > 1")
	}
	return nil
}
