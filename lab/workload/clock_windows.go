package workload

import (
	"syscall"
	"unsafe"
)

// On Windows time.Now advances in ~0.5ms steps (assume.TestClockResolution),
// far too coarse to time one call, so the lab clock reads the performance
// counter instead.
var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	qpc      = kernel32.NewProc("QueryPerformanceCounter")
	qpcFreq  = func() int64 {
		var f int64
		kernel32.NewProc("QueryPerformanceFrequency").Call(uintptr(unsafe.Pointer(&f)))
		return f
	}()
)

// Now reads the lab clock.
func Now() Stamp {
	var c int64
	qpc.Call(uintptr(unsafe.Pointer(&c)))
	return ticksToNanos(c, qpcFreq)
}
