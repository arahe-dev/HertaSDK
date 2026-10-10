package doe

import (
	"runtime/metrics"
	"sync"
	"sync/atomic"
	"time"
)

// sampler tracks peak goroutine count and peak live heap, sampled every
// millisecond through runtime/metrics (no stop-the-world).
type sampler struct {
	peakG, peakHeap atomic.Uint64
	stop            chan struct{}
	once            sync.Once
	wg              sync.WaitGroup
	samples         []metrics.Sample
}

func startSampler() *sampler {
	s := &sampler{stop: make(chan struct{}), samples: []metrics.Sample{
		{Name: "/sched/goroutines:goroutines"},
		{Name: "/memory/classes/heap/objects:bytes"},
	}}
	s.wg.Add(1)
	go s.loop()
	return s
}

func (s *sampler) loop() {
	defer s.wg.Done()
	t := time.NewTicker(time.Millisecond)
	defer t.Stop()
	for {
		s.sample()
		select {
		case <-t.C:
		case <-s.stop:
			return
		}
	}
}

func (s *sampler) sample() {
	metrics.Read(s.samples)
	raise(&s.peakG, s.samples[0].Value.Uint64())
	raise(&s.peakHeap, s.samples[1].Value.Uint64())
}

// reset restarts peak tracking from zero (the next sample sets it).
func (s *sampler) reset() { s.peakG.Store(0); s.peakHeap.Store(0) }

// close stops sampling; it is idempotent.
func (s *sampler) close() { s.once.Do(func() { close(s.stop) }); s.wg.Wait() }

func raise(a *atomic.Uint64, v uint64) {
	for {
		old := a.Load()
		if v <= old || a.CompareAndSwap(old, v) {
			return
		}
	}
}

// liveGoroutines reads the same metric the sampler peaks on, so baselines
// and peaks count goroutines the same way.
func liveGoroutines() uint64 {
	s := []metrics.Sample{{Name: "/sched/goroutines:goroutines"}}
	metrics.Read(s)
	return s[0].Value.Uint64()
}
