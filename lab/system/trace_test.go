package system_test

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// opSpec is an operation as the trace needs to know it.
type opSpec struct {
	name   string
	policy impl.Policy
	key    func(int) string // nil: unkeyed
}

// retryable reports whether the op may retry an attempt that returned o
// (§5 step 5): requested by Retry.On, and safe for the Effect.
func (o opSpec) retryable(oc impl.Outcome) bool {
	switch oc {
	case impl.Transient:
		return o.policy.On[impl.Transient]
	case impl.Uncertain:
		return o.policy.On[impl.Uncertain] && o.policy.Effect != impl.NonIdempotent
	}
	return false
}

// liveCall is the trace of one Do that has not returned yet.
type liveCall struct {
	op       int
	attempts int
	mayRetry bool // the last attempt returned an outcome the op may retry
	panicked bool // the last attempt panicked
	issued   workload.Stamp
}

// tracer enforces, from inside every handler, the safety invariants —
// capacity, key exclusion, "never retry what may not be retried" — and
// folds each finished call into a ledger. Memory is bounded by the calls
// in flight, so a soak measures the runtime, not the lab.
type tracer struct {
	ops     []opSpec
	mon     *workload.Monitor
	running atomic.Int64 // handler attempts in flight

	mu      sync.Mutex
	next    int
	live    map[int]*liveCall
	ledger  ledger
	waits   map[int]*reservoir // per op: Do → first handler entry
	unsafe  []string
	panics  []string
	nunsafe int
}

// ledger is the per-call trace, aggregated.
type ledger struct {
	calls, overloads, stopped, entered, retries uint64
	classes                                     map[impl.Class]int
}

func newTracer(specs []impl.Spec, ops []opSpec) *tracer {
	return &tracer{ops: ops, mon: workload.NewMonitor(specs...), live: map[int]*liveCall{},
		ledger: ledger{classes: map[impl.Class]int{}}, waits: map[int]*reservoir{}}
}

// issue registers a new call of op and returns its id (the Do input).
func (tr *tracer) issue(op int) int {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	id := tr.next
	tr.next++
	tr.live[id] = &liveCall{op: op, issued: workload.Now()}
	return id
}

// enter starts an attempt of call id. The handler must defer exit with
// the outcome it returned (Success for nil, Permanent for untagged) and
// whether it returned at all (false: it is panicking).
func (tr *tracer) enter(id int) (attempt int, exit func(returned impl.Outcome, returnedNormally bool)) {
	tr.running.Add(1)
	now := workload.Now()
	tr.mu.Lock()
	c := tr.live[id]
	op := tr.ops[c.op]
	if c.attempts > 0 && !c.mayRetry {
		if tr.nunsafe++; len(tr.unsafe) < 3 {
			tr.unsafe = append(tr.unsafe, fmt.Sprintf("%s call %d: attempt %d after a non-retryable attempt", op.name, id, c.attempts+1))
		}
	}
	if c.attempts == 0 {
		w := tr.waits[c.op]
		if w == nil {
			w = &reservoir{}
			tr.waits[c.op] = w
		}
		w.add(time.Duration(now - c.issued))
	}
	c.attempts++
	attempt = c.attempts
	tr.mu.Unlock()
	key := ""
	if op.key != nil {
		key = op.key(id)
	}
	leave := tr.mon.Enter(op.policy.Resources, key)
	return attempt, func(returned impl.Outcome, returnedNormally bool) {
		leave()
		tr.mu.Lock()
		c.mayRetry, c.panicked = returnedNormally && op.retryable(returned), !returnedNormally
		tr.mu.Unlock()
		tr.running.Add(-1)
	}
}

// finish folds call id into the ledger; caught says the caller saw a panic.
func (tr *tracer) finish(id int, class impl.Class, caught bool) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	c := tr.live[id]
	delete(tr.live, id)
	l := &tr.ledger
	l.calls++
	l.classes[class]++
	switch class {
	case impl.ClassOverloaded:
		l.overloads++
	case impl.ClassStopping:
		l.stopped++
	}
	if c.attempts > 0 {
		l.entered++
		l.retries += uint64(c.attempts - 1)
	}
	if c.panicked != caught && len(tr.panics) < 3 {
		tr.panics = append(tr.panics, fmt.Sprintf("call %d: handler panicked=%v, caller saw a panic=%v (§5.1)", id, c.panicked, caught))
	}
}

// check fails t on any safety violation and reconciles Stats, taken after
// a successful drain, with the ledger.
func (tr *tracer) check(t *testing.T, im impl.Impl, st impl.Stats) {
	t.Helper()
	tr.mon.Check(t)
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if tr.nunsafe > 0 {
		t.Errorf("%d unsafe retries, first: %v", tr.nunsafe, tr.unsafe)
	}
	if len(tr.panics) > 0 {
		t.Errorf("panic propagation: %v", tr.panics)
	}
	if len(tr.live) > 0 {
		t.Errorf("%d calls never returned", len(tr.live))
	}
	l := tr.ledger
	// Rejected = gate refusals + overloads. v0.2 also rewrites an overload into
	// ErrRuntimeStopping when Shutdown began after the gate (admissionError),
	// so some counted overloads surface as "stopping" (D11: a race in v0.2).
	masked := int64(st.Rejected) - int64(l.calls-st.Admitted) - int64(l.overloads)
	if masked != 0 {
		t.Logf("%d overloads were reported as ErrRuntimeStopping", masked)
	}
	switch {
	case st.Finished != st.Admitted:
		t.Errorf("drained, yet Finished %d != Admitted %d", st.Finished, st.Admitted)
	case masked < 0 || masked > int64(l.stopped) || (masked > 0 && workload.Fixed(im, workload.D11)):
		t.Errorf("Rejected %d != gate refusals %d + overloads %d (%d overloads reported as stopping)", st.Rejected, l.calls-st.Admitted, l.overloads, masked)
	case st.Retried != l.retries:
		t.Errorf("Retried %d, handlers saw %d retries", st.Retried, l.retries)
	case workload.Fixed(im, workload.D9) && st.Started != l.entered:
		t.Errorf("Started %d, handlers entered by %d calls (D9)", st.Started, l.entered)
	case st.Started < l.entered || st.Started > st.Admitted:
		t.Errorf("Started %d outside [entered %d, Admitted %d]", st.Started, l.entered, st.Admitted)
	}
}

// summary is the ledger in one line.
func (tr *tracer) summary() string {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return fmt.Sprintf("%d calls %v", tr.ledger.calls, tr.ledger.classes)
}

// admissionWaits is the Do → first-handler-entry distribution of the ops.
func (tr *tracer) admissionWaits(ops ...int) workload.Dist {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	var ds []time.Duration
	for _, op := range ops {
		if w := tr.waits[op]; w != nil {
			ds = append(ds, w.samples...)
		}
	}
	return workload.Summarize(ds)
}

// reservoir keeps a uniform sample of at most 50 000 durations.
type reservoir struct {
	n       int
	samples []time.Duration
}

func (r *reservoir) add(d time.Duration) {
	r.n++
	switch {
	case len(r.samples) < 50_000:
		r.samples = append(r.samples, d)
	default:
		if j := rand.IntN(r.n); j < len(r.samples) {
			r.samples[j] = d
		}
	}
}
