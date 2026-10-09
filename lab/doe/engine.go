package doe

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// drainLimit bounds Shutdown and the callers' exit after it; beyond it a run
// is reported as hung rather than stalling the experiment.
const drainLimit = 5 * time.Second

// engine is one run in progress.
type engine struct {
	w     Workload
	im    impl.Impl
	rt    impl.Runtime
	ops   [2]impl.Op // light (1 unit per budget), heavy (HeavyUnits)
	keys  []string
	slots []slot
	mon   monitor
	rec   *recorder
	epoch int64 // nanotime at build

	from, to atomic.Int64 // measurement window [from, to); to == 0 while open
	stop     atomic.Bool
	dropped  atomic.Uint64
}

// now is the run clock: monotonic ns since the run began, ≥ 1 so 0 can mean
// "not yet".
func (e *engine) now() int64 { return nanotime() - e.epoch + 1 }

func (e *engine) inWindow(t0, t1 int64) bool {
	to := e.to.Load()
	return t0 >= e.from.Load() && (to == 0 || t1 < to)
}

// Run executes one workload against one implementation: build, warm up,
// measure, then shut down under load and check the drain.
func Run(im impl.Impl, w Workload) Result {
	if err := w.validate(); err != nil {
		return failed(err)
	}
	e, err := build(im, w)
	if err != nil {
		return failed(err)
	}
	runtime.GC()
	baseG := runtime.NumGoroutine()
	smp := startSampler()
	defer smp.close()
	idleG := liveGoroutines() // includes the sampler
	e.from.Store(math.MaxInt64)
	callers, harnessG := e.launch()

	time.Sleep(w.Warmup)
	var m0, m1 runtime.MemStats
	runtime.ReadMemStats(&m0)
	s0 := e.rt.Stats()
	smp.reset()
	from := e.now()
	e.from.Store(from)
	time.Sleep(w.Duration)
	to := e.now()
	e.to.Store(to)
	runtime.ReadMemStats(&m1)
	s1 := e.rt.Stats()
	peakG, peakHeap := smp.peakG.Load(), smp.peakHeap.Load()

	var r Result
	if !e.shutdown(&r, callers) {
		return r // callers are stuck inside Do; their records are unsafe to read
	}
	e.collect(&r, float64(to-from)/1e9)
	ops := float64(r.Ops)
	r.AllocsPerOp = float64(m1.Mallocs-m0.Mallocs) / ops
	r.BytesPerOp = float64(m1.TotalAlloc-m0.TotalAlloc) / ops
	r.GCPerSec = float64(m1.NumGC-m0.NumGC) / r.Seconds
	r.GCPauseUs = float64(m1.PauseTotalNs-m0.PauseTotalNs) / 1e3
	r.RetriesPerOp = float64(s1.Retried-s0.Retried) / ops
	r.PeakGoroutines = max(0, float64(peakG)-float64(idleG)-float64(harnessG))
	r.PeakHeapMB = float64(peakHeap) / (1 << 20)

	smp.close()
	time.Sleep(50 * time.Millisecond)
	r.Residual = runtime.NumGoroutine() - baseG
	return r
}

// shutdown calls Shutdown while the callers are still issuing calls, records
// the drain, then stops the callers. It reports false if they hang.
func (e *engine) shutdown(r *Result, callers *sync.WaitGroup) bool {
	ctx, cancel := context.WithTimeout(context.Background(), drainLimit)
	defer cancel()
	t := nanotime()
	err := e.rt.Shutdown(ctx)
	r.DrainMs = float64(nanotime()-t) / 1e6
	if err != nil {
		r.DrainErr = err.Error()
	} else if st := e.rt.Stats(); st.Admitted > st.Finished {
		r.StatsLag = st.Admitted - st.Finished
	}
	e.stop.Store(true)
	done := make(chan struct{})
	go func() { callers.Wait(); close(done) }()
	select {
	case <-done:
		return true
	case <-time.After(drainLimit):
		*r = failed(fmt.Errorf("callers hung %v after Shutdown", drainLimit))
		r.Hung = true
		return false
	}
}

// collect merges the recorder and the monitors into r.
func (e *engine) collect(r *Result, secs float64) {
	lat, wait, exec, all, ok, errs := e.rec.totals()
	r.Seconds, r.Ops, r.OK, r.Errors = secs, all, ok, errs
	r.Throughput, r.Goodput = float64(all)/secs, float64(ok)/secs
	r.Lat, r.Wait, r.Exec = quantiles(&lat), quantiles(&wait), quantiles(&exec)
	r.CapViolations, r.KeyViolations = e.mon.capViol.Load(), e.mon.keyViol.Load()
	r.Jain = math.NaN()
	if e.w.Rate == 0 {
		done := make([]float64, len(e.slots))
		for i := range e.slots {
			done[i] = float64(e.slots[i].done)
		}
		r.Jain = jain(done)
	} else if d := float64(e.dropped.Load()); d+float64(all) > 0 {
		r.DropRate = d / (d + float64(all))
	}
}

func failed(err error) Result {
	nan := math.NaN()
	q := Quantiles{nan, nan, nan, nan, nan, nan}
	return Result{Err: err.Error(), Lat: q, Wait: q, Exec: q, Jain: nan, Seconds: nan,
		Throughput: nan, Goodput: nan, AllocsPerOp: nan, BytesPerOp: nan, GCPerSec: nan,
		GCPauseUs: nan, PeakGoroutines: nan, PeakHeapMB: nan, RetriesPerOp: nan,
		DropRate: nan, DrainMs: nan}
}

// build constructs the runtime, the light and heavy operations, and the
// per-run state.
func build(im impl.Impl, w Workload) (*engine, error) {
	e := &engine{w: w, im: im, epoch: nanotime()}
	specs := make([]impl.Spec, w.Resources)
	for i := range specs {
		specs[i] = impl.Spec{Name: fmt.Sprintf("r%d", i), Capacity: w.Capacity}
	}
	rt, err := im.NewRuntime(specs...)
	if err != nil {
		return nil, fmt.Errorf("NewRuntime: %w", err)
	}
	e.rt = rt
	p := impl.Policy{Effect: impl.Idempotent, Admission: w.Admission, Timeout: w.Timeout, MaxAttempts: w.MaxAttempts}
	if w.MaxAttempts > 1 {
		p.On = map[impl.Outcome]bool{impl.Transient: true, impl.Uncertain: true}
	}
	if w.Keys > 0 {
		e.keys = make([]string, w.Keys)
		for i := range e.keys {
			e.keys[i] = fmt.Sprintf("k%d", i)
		}
		p.Key = func(in int) string { return e.keys[e.slots[in].key] }
	}
	for i, units := range [2]int64{1, w.heavyUnits()} {
		p.Resources = nil
		for _, s := range specs {
			p.Resources = append(p.Resources, impl.Req{Name: s.Name, Units: units})
		}
		if e.ops[i], err = rt.NewOp(fmt.Sprintf("op%d", i), p, e.handle); err != nil {
			return nil, fmt.Errorf("NewOp: %w", err)
		}
	}
	e.slots = make([]slot, w.Clients)
	e.mon = monitor{
		capacity: w.Capacity,
		inUse:    make([]atomic.Int64, w.Resources),
		holders:  make([]atomic.Int32, max(w.Keys, 1)),
	}
	e.rec = newRecorder(w.Clients)
	return e, nil
}
