package workload

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
)

// ErrScripted is the error scripted handlers return or panic with.
var ErrScripted = errors.New("lab: scripted failure")

// Rig drives one implementation through a model.Scenario. It must be built
// and used inside a testing/synctest bubble: View settles every goroutine
// with synctest.Wait, so what it reports is exact, with no sleeps and no
// polling. A call that is live but not inside a held handler attempt is,
// by construction, blocked inside the implementation: Parked.
type Rig struct {
	im  impl.Impl
	sc  *model.Scenario
	rt  impl.Runtime
	ops []impl.Op

	mu    sync.Mutex
	calls []*rigCall
	downs []*model.ShutdownView
}

type rigCall struct {
	ctx      context.Context
	cancel   context.CancelFunc
	release  chan struct{}
	done     chan struct{}
	released bool
	started  bool
	holding  bool
	finished bool
	res      model.Result
	elapsed  time.Duration
}

// NewRig builds sc's runtime and operations on im.
func NewRig(im impl.Impl, sc *model.Scenario) (*Rig, error) {
	rt, err := im.NewRuntime(sc.Specs...)
	if err != nil {
		return nil, fmt.Errorf("NewRuntime: %w", err)
	}
	r := &Rig{im: im, sc: sc, rt: rt}
	for _, op := range sc.Ops {
		o, err := rt.NewOp(op.Name, op.Policy(r.key), r.handle)
		if err != nil {
			return nil, fmt.Errorf("NewOp %s: %w", op.Name, err)
		}
		r.ops = append(r.ops, o)
	}
	for range sc.Calls {
		r.calls = append(r.calls, &rigCall{release: make(chan struct{}), done: make(chan struct{})})
	}
	return r, nil
}

// Runtime exposes the runtime under test.
func (r *Rig) Runtime() impl.Runtime { return r.rt }

func (r *Rig) key(in int) string { return r.sc.Calls[in].Key }

// Apply triggers one step; it does not wait for it to settle.
func (r *Rig) Apply(st model.Step) {
	switch st.Kind {
	case model.StepStart:
		r.Start(st.Call)
	case model.StepRelease:
		r.Release(st.Call)
	case model.StepCancel:
		r.calls[st.Call].cancel()
	case model.StepShutdown:
		r.Shutdown(st.Expired)
	}
}

// Start calls Do for call i on its own goroutine.
func (r *Rig) Start(i int) {
	spec, c := r.sc.Calls[i], r.calls[i]
	parent, stop := context.Background(), context.CancelFunc(func() {})
	if spec.Deadline > 0 {
		parent, stop = context.WithTimeout(parent, spec.Deadline)
	}
	c.ctx, c.cancel = context.WithCancel(parent)
	if spec.Precancel {
		c.cancel()
	}
	r.mu.Lock()
	c.started = true
	r.mu.Unlock()
	op := r.ops[spec.Op]
	go func() {
		defer close(c.done)
		defer stop()
		start := time.Now()
		res := r.do(op, c.ctx, i)
		r.mu.Lock()
		res.Calls = c.res.Calls
		c.res, c.finished, c.elapsed = res, true, time.Since(start)
		r.mu.Unlock()
	}()
}

func (r *Rig) do(op impl.Op, ctx context.Context, in int) (res model.Result) {
	defer func() {
		if recover() != nil {
			res = model.Result{Panic: true}
		}
	}()
	v, err := op.Do(ctx, in)
	res = model.Result{Class: r.im.Classify(err), Outcome: r.im.OutcomeOf(err)}
	if err == nil {
		res.Value = v
	}
	return res
}

// Release lets call i's held attempt return.
func (r *Rig) Release(i int) {
	c := r.calls[i]
	r.mu.Lock()
	defer r.mu.Unlock()
	if !c.released {
		c.released = true
		close(c.release)
	}
}

// Cancel cancels call i's context.
func (r *Rig) Cancel(i int) { r.calls[i].cancel() }

// Shutdown calls Shutdown on its own goroutine, with a done ctx if expired.
func (r *Rig) Shutdown(expired bool) {
	ctx, cancel := context.WithCancel(context.Background())
	if expired {
		cancel()
	}
	sd := &model.ShutdownView{}
	r.mu.Lock()
	r.downs = append(r.downs, sd)
	r.mu.Unlock()
	go func() {
		defer cancel()
		err := r.rt.Shutdown(ctx)
		st := r.rt.Stats()
		r.mu.Lock()
		sd.Returned, sd.Class, sd.Lag = true, r.im.Classify(err), err == nil && st.Finished < st.Admitted
		r.mu.Unlock()
	}()
}

// Await blocks until call i returns, letting virtual time advance.
func (r *Rig) Await(i int) model.Result {
	<-r.calls[i].done
	return r.Result(i)
}

// Result is call i's result so far.
func (r *Rig) Result(i int) model.Result {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[i].res
}

// Elapsed is the virtual duration of call i's Do.
func (r *Rig) Elapsed(i int) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls[i].elapsed
}

// View settles the bubble and reports what is observable.
func (r *Rig) View() model.View {
	waitIdle()
	r.mu.Lock()
	defer r.mu.Unlock()
	v := model.View{Stats: r.rt.Stats()}
	for _, c := range r.calls {
		p := model.Parked
		switch {
		case !c.started:
			p = model.Idle
		case c.finished:
			p = model.Done
		case c.holding:
			p = model.Holding
		}
		v.Phases = append(v.Phases, p)
		v.Results = append(v.Results, c.res)
	}
	for _, sd := range r.downs {
		v.Shutdowns = append(v.Shutdowns, *sd)
	}
	return v
}

// Close shuts the runtime down, releases every holder, and reports any call
// that did not finish or a Stats ledger that does not balance.
func (r *Rig) Close() error {
	r.Shutdown(false)
	for i := range r.calls {
		r.Release(i)
	}
	v := r.View()
	for i, p := range v.Phases {
		if p != model.Idle && p != model.Done {
			return fmt.Errorf("c%d still %v after Shutdown and release", i, p)
		}
	}
	if s := v.Stats; s.Admitted != s.Finished || s.Started > s.Admitted {
		return fmt.Errorf("unbalanced stats after drain: %+v", s)
	}
	return nil
}

// handle is every operation's handler: it plays the call's script.
func (r *Rig) handle(ctx context.Context, in int) (int, error) {
	c := r.calls[in]
	r.mu.Lock()
	c.res.Calls++
	a := r.sc.Calls[in].Script.At(c.res.Calls)
	r.mu.Unlock()

	switch a.Block {
	case model.BlockHold:
		r.hold(c, func() { <-c.release })
	case model.BlockHoldCtx:
		ended := false
		r.hold(c, func() {
			select {
			case <-c.release:
			case <-ctx.Done():
				ended = true
			}
		})
		if ended {
			return in, ctx.Err()
		}
	case model.BlockDeadline:
		<-ctx.Done()
	case model.BlockSleep:
		time.Sleep(a.Sleep)
	}
	if a.Cancel {
		c.cancel()
	}
	switch a.Return {
	case model.ReturnPlain:
		return 0, ErrScripted
	case model.ReturnCtxErr:
		return in, ctx.Err()
	case model.ReturnTag:
		return 0, r.im.Fail(a.Outcome, ErrScripted)
	case model.ReturnPanic:
		panic(ErrScripted)
	}
	return in, nil
}

func (r *Rig) hold(c *rigCall, block func()) {
	r.mu.Lock()
	c.holding = true
	r.mu.Unlock()
	block()
	r.mu.Lock()
	c.holding = false
	r.mu.Unlock()
}
