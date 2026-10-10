//go:build !windows

package workload

import "time"

var epoch = time.Now()

// Now reads the lab clock.
func Now() Stamp { return ticksToNanos(int64(time.Since(epoch)), 1e9) }
