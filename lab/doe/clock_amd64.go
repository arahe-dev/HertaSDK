package doe

// ticks reads the time-stamp counter (LFENCE; RDTSC). Modern x86 CPUs keep
// it invariant (constant rate, synchronised across cores).
func ticks() uint64
