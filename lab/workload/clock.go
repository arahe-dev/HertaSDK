package workload

import "time"

// Stamp is a monotonic timestamp from the lab clock.
type Stamp int64

// Since is the time elapsed from s.
func Since(s Stamp) time.Duration { return time.Duration(Now() - s) }

// ticksToNanos converts counter ticks at freq Hz to nanoseconds.
func ticksToNanos(ticks, freq int64) Stamp {
	return Stamp(ticks/freq*1e9 + ticks%freq*1e9/freq)
}
