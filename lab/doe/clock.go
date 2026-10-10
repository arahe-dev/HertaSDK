package doe

import (
	"sync"
	"time"
)

// The lab's clock. Go's time.Now on Windows advances in ~0.6ms steps (it
// reads the interrupt time), which would quantise every latency and every
// busy-spin in this package. On amd64 the lab reads the invariant TSC
// instead, calibrated once against the OS clock; elsewhere it falls back to
// time.Since.

var (
	clockOnce   sync.Once
	nsPerTick   float64
	clockOrigin uint64
)

// nanotime is a monotonic, high-resolution clock in nanoseconds.
func nanotime() int64 {
	clockOnce.Do(calibrate)
	return int64(float64(ticks()-clockOrigin) * nsPerTick)
}

// calibrate times ticks across a span bounded by two edges of the coarse OS
// clock: at an edge the OS clock's value is exact, so the error is only the
// few nanoseconds it takes to notice the edge.
func calibrate() {
	const span = 50 * time.Millisecond
	t0, k0 := edge()
	for time.Since(t0) < span {
	}
	t1, k1 := edge()
	nsPerTick = float64(t1.Sub(t0)) / float64(k1-k0)
	clockOrigin = k0
}

// edge spins until the OS clock changes and returns the new reading with
// the tick count observed at that moment.
func edge() (time.Time, uint64) {
	last := time.Now()
	for {
		now, k := time.Now(), ticks()
		if now != last {
			return now, k
		}
	}
}
