package workload

import (
	"bytes"
	"runtime"
	"testing"
	"time"
)

// Goroutines waits up to d for the goroutine count to fall to at most
// want, and returns the last count seen. Exited goroutines are reaped
// asynchronously, so a single read right after a drain overcounts.
func Goroutines(want int, d time.Duration) int {
	deadline := time.Now().Add(d)
	for {
		n := runtime.NumGoroutine()
		if n <= want || time.Now().After(deadline) {
			return n
		}
		time.Sleep(time.Millisecond)
	}
}

// NoLeak fails t if the goroutine count has not returned to base (plus
// slack) within d, and dumps the survivors.
func NoLeak(t testing.TB, base, slack int, d time.Duration) {
	t.Helper()
	if n := Goroutines(base+slack, d); n > base+slack {
		t.Errorf("goroutine leak: %d live, baseline %d (+%d slack)\n%s", n, base, slack, stacks())
	}
}

// LiveHeap is the heap in use once goroutines have settled and two forced
// collections have run.
func LiveHeap() uint64 {
	SettledGoroutines() // a goroutine still exiting keeps its closure live
	runtime.GC()
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.HeapAlloc
}

// Within runs fn and fails t, with every goroutine's stack, if fn has not
// returned after d. It is the lab's hang detector.
func Within(t testing.TB, d time.Duration, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s did not finish within %v\n%s", what, d, stacks())
	}
}

func stacks() []byte {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	if len(buf) > 32<<10 {
		buf = append(buf[:32<<10:32<<10], []byte("\n... (truncated)")...)
	}
	return bytes.TrimSpace(buf)
}

// SettledGoroutines waits until the goroutine count stops changing (no
// change over 5 consecutive 1ms samples, at most 1s) and returns it: a
// baseline taken while exited goroutines are still being reaped is wrong.
func SettledGoroutines() int {
	n, stable := runtime.NumGoroutine(), 0
	for deadline := time.Now().Add(time.Second); stable < 5 && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
		if m := runtime.NumGoroutine(); m == n {
			stable++
		} else {
			n, stable = m, 0
		}
	}
	return n
}
