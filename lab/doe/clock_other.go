//go:build !amd64

package doe

import "time"

var tickBase = time.Now()

// ticks falls back to the OS monotonic clock, in nanoseconds.
func ticks() uint64 { return uint64(time.Since(tickBase)) }
