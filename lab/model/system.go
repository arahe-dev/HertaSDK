package model

import (
	"fmt"
	"slices"
	"strings"

	"github.com/arahe-dev/hertasdk/lab/impl"
)

// System composes the subsystem models with admission and drain.
type System struct {
	rules     Rules
	sc        *Scenario
	sems      map[string]*Sem
	keys      Keys
	calls     []*sysCall
	stopping  bool
	stats     impl.Stats
	shutdowns []ShutdownView
	expired   []bool

	root   int          // the call driving the current step (-1: none)
	active map[int]bool // woken calls that acted concurrently with root
	woken  int          // grants made to waiters during the current step
}

type sysCall struct {
	phase  Phase
	call   Call
	op     Op
	claims []impl.Req // in global (name) order
	next   int        // claims acquired so far
	hasKey bool
	exec   *Exec
	res    Result
}

// NewSystem builds the model of sc's runtime under rules.
func NewSystem(r Rules, sc *Scenario) *System {
	s := &System{rules: r, sc: sc, sems: map[string]*Sem{}}
	for _, sp := range sc.Specs {
		s.sems[sp.Name] = &Sem{Cap: sp.Capacity}
	}
	for i, c := range sc.Calls {
		op := sc.Ops[c.Op]
		claims := slices.Clone(op.Resources)
		slices.SortFunc(claims, func(a, b impl.Req) int { return strings.Compare(a.Name, b.Name) })
		s.calls = append(s.calls, &sysCall{call: c, op: op, claims: claims, exec: NewExec(r, op, c, i)})
	}
	return s
}

// Check replays sc against the model and reports the first step that is
// ambiguous or not applicable.
func Check(r Rules, sc *Scenario) error {
	s := NewSystem(r, sc)
	for i, st := range sc.Steps {
		if err := s.Apply(st); err != nil {
			return fmt.Errorf("step %d %v: %w", i, st, err)
		}
	}
	return nil
}

// Apply performs one step. On error the System is no longer usable.
func (s *System) Apply(st Step) error {
	s.root, s.active, s.woken = st.Call, map[int]bool{}, 0
	switch st.Kind {
	case StepStart:
		if s.calls[st.Call].phase != Idle {
			return fmt.Errorf("model: c%d already started", st.Call)
		}
		s.start(st.Call)
	case StepRelease:
		c := s.calls[st.Call]
		if c.phase != Holding {
			return fmt.Errorf("model: c%d is not holding", st.Call)
		}
		c.exec.Release()
		s.settle(st.Call)
	case StepCancel:
		if err := s.cancel(st.Call); err != nil {
			return err
		}
	case StepShutdown:
		s.root = -1
		if err := s.shutdown(st.Expired); err != nil {
			return err
		}
	}
	if len(s.active) > 1 {
		return fmt.Errorf("%w: %d woken calls race", ErrAmbiguous, len(s.active))
	}
	s.drainShutdowns()
	return nil
}

func (s *System) start(id int) {
	if s.stopping { // §4 step 1
		s.stats.Rejected++
		c := s.calls[id]
		c.phase, c.res = Done, Result{Class: impl.ClassStopping, Outcome: impl.Permanent}
		return
	}
	s.stats.Admitted++
	s.advance(id)
}

// advance continues admission (§4 steps 2-3) from wherever the call is.
func (s *System) advance(id int) {
	c := s.calls[id]
	reject := c.op.Admission == impl.Reject
	if key := c.call.Key; c.op.Keyed && key != "" && !c.hasKey {
		switch {
		case c.exec.CallerErr() != impl.ClassNil: // only a live caller takes a key (Amendment 1)
			s.finish(id, Result{Class: c.exec.CallerErr(), Outcome: impl.Permanent})
			return
		case reject && s.rules.Fixed: // Δ1: fail fast on a held key
			if s.keys.Held(key) {
				if KeyRejectCounted {
					s.stats.Rejected++
				}
				s.finish(id, overloaded)
				return
			}
			c.hasKey = s.keys.Acquire(id, key)
		case s.keys.Acquire(id, key):
			c.hasKey = true
		default:
			c.phase = Parked
			return
		}
	}
	for c.next < len(c.claims) {
		cl := c.claims[c.next]
		sem := s.sems[cl.Name]
		switch {
		case reject:
			if !sem.TryAcquire(cl.Units) {
				s.stats.Rejected++
				s.finish(id, overloaded)
				return
			}
		case c.exec.CallerErr() != impl.ClassNil:
			s.finish(id, Result{Class: c.exec.CallerErr(), Outcome: impl.Permanent})
			return
		case !sem.Acquire(id, cl.Units):
			c.phase = Parked
			return
		}
		c.next++
	}
	// Δ9: v0.2 counts Started once everything is acquired; the contract
	// counts it when the handler is entered for the first time.
	if !s.rules.Fixed {
		s.stats.Started++
	}
	c.exec.Run()
	if s.rules.Fixed && c.exec.Calls > 0 {
		s.stats.Started++
	}
	s.settle(id)
}

var overloaded = Result{Class: impl.ClassOverloaded, Outcome: impl.Throttled}

// settle records a call that is holding or has returned from its handler.
func (s *System) settle(id int) {
	c := s.calls[id]
	if !c.exec.Done {
		c.phase = Holding
		return
	}
	s.finish(id, c.exec.Res)
}

// finish ends an admitted call: count, then release in reverse order.
func (s *System) finish(id int, r Result) {
	c := s.calls[id]
	c.phase, c.res = Done, r
	s.stats.Finished++
	s.stats.Retried += uint64(c.exec.Retried)
	n := c.next
	c.next = 0
	for i := n - 1; i >= 0; i-- {
		for _, w := range s.sems[c.claims[i].Name].Release(c.claims[i].Units) {
			s.wake(w, false)
		}
	}
	if c.hasKey {
		c.hasKey = false
		if w := s.keys.Release(c.call.Key); w >= 0 {
			s.wake(w, true)
		}
	}
}

// wake continues a waiter that was just granted its key or next claim. A
// woken call that does anything but enter a held attempt is "active": it
// runs concurrently with the step's root, and two of those can race.
func (s *System) wake(id int, key bool) {
	s.woken++
	c := s.calls[id]
	if key {
		c.hasKey = true
	} else {
		c.next++
	}
	more := (key && len(c.claims) > 0) || (!key && c.next < len(c.claims))
	s.advance(id)
	if id != s.root && (more || c.phase != Holding) {
		s.active[id] = true
	}
}

func (s *System) cancel(id int) error {
	c := s.calls[id]
	switch c.phase {
	case Parked:
		c.exec.Cancel()
		s.withdraw(id, Result{Class: c.exec.CallerErr(), Outcome: impl.Permanent})
	case Holding:
		c.exec.Cancel()
		s.settle(id)
	default:
		return fmt.Errorf("model: c%d is not live", id)
	}
	return nil
}

// withdraw fails a parked call: it leaves its queue, then releases.
func (s *System) withdraw(id int, r Result) {
	c := s.calls[id]
	if c.op.Keyed && c.call.Key != "" && !c.hasKey {
		s.keys.Cancel(id, c.call.Key)
	} else {
		for _, w := range s.sems[c.claims[c.next].Name].Cancel(id) {
			s.wake(w, false)
		}
	}
	s.finish(id, r)
}

// shutdown stops admission and withdraws every waiter (§6). A waiter that
// holds something, or whose withdrawal grants another waiter, hands work to
// a call that is itself racing the stop signal (§8: v0.2 nondeterminism),
// so both states are ambiguous.
func (s *System) shutdown(expired bool) error {
	if !s.stopping {
		for _, c := range s.calls {
			if c.phase == Parked && (c.next > 0 || c.hasKey) {
				return fmt.Errorf("%w: shutdown while a waiter holds units", ErrAmbiguous)
			}
		}
		s.stopping = true
		for id, c := range s.calls {
			if c.phase == Parked {
				s.withdraw(id, Result{Class: impl.ClassStopping, Outcome: impl.Permanent})
			}
		}
		if s.woken > 0 {
			return fmt.Errorf("%w: a withdrawal granted a waiter mid-shutdown", ErrAmbiguous)
		}
	}
	s.shutdowns = append(s.shutdowns, ShutdownView{})
	s.expired = append(s.expired, expired)
	return nil
}

// drainShutdowns resolves Shutdown calls once nothing admitted is live.
func (s *System) drainShutdowns() {
	drained := true
	for _, c := range s.calls {
		if c.phase == Parked || c.phase == Holding {
			drained = false
		}
	}
	for i := range s.shutdowns {
		sd := &s.shutdowns[i]
		switch {
		case sd.Returned:
		case drained:
			sd.Returned = true
			// v0.2 races "drained" against "expired" (§6, nondeterministic).
			sd.Either = s.expired[i] && !s.rules.Fixed
		case s.expired[i]:
			sd.Returned, sd.Class = true, impl.ClassCanceled
		}
	}
}

// View is what a black-box observer sees once the last step settled.
func (s *System) View() View {
	v := View{Stats: s.stats, Shutdowns: slices.Clone(s.shutdowns)}
	for _, c := range s.calls {
		v.Phases = append(v.Phases, c.phase)
		v.Results = append(v.Results, c.res)
	}
	return v
}

// Elapsed is the virtual duration of call id's Do (calls without holds).
func (s *System) Elapsed(id int) int64 { return int64(s.calls[id].exec.Now()) }
