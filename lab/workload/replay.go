package workload

import (
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
)

func waitIdle() { synctest.Wait() }

// Divergence is the first point where an implementation left its model.
type Divergence struct {
	Step int // -1: construction or teardown
	What string
}

func (d *Divergence) Error() string { return fmt.Sprintf("step %d: %s", d.Step, d.What) }

// Replay runs sc on im in a fresh synctest bubble and compares the
// observable View with im's model after every step. It reports, rather
// than fails, so minimizers can call it repeatedly.
func Replay(t *testing.T, im impl.Impl, sc model.Scenario) *Divergence {
	return ReplayAs(t, im, Rules(im), sc)
}

// ReplayAs is Replay against an explicit model mode. Judging v0.2 by the
// fixed rules is how the lab proves it can see the §8 defects at all.
func ReplayAs(t *testing.T, im impl.Impl, rules model.Rules, sc model.Scenario) (div *Divergence) {
	RequireBubbles(t, im)
	synctest.Test(t, func(t *testing.T) {
		sys := model.NewSystem(rules, &sc)
		rig, err := NewRig(im, &sc)
		if err != nil {
			div = &Divergence{-1, err.Error()}
			return
		}
		defer func() {
			if err := rig.Close(); err != nil && div == nil {
				div = &Divergence{-1, err.Error()}
			}
		}()
		for i, st := range sc.Steps {
			if err := sys.Apply(st); err != nil {
				div = &Divergence{i, err.Error()}
				return
			}
			rig.Apply(st)
			if d := model.Diff(rules, sys.View(), rig.View()); d != "" {
				div = &Divergence{i, fmt.Sprintf("%v: %s", st, d)}
				return
			}
		}
	})
	return div
}

// Bubble runs fn with a fresh rig for sc inside a synctest bubble and
// closes the rig afterwards, failing t on construction or drain errors.
func Bubble(t *testing.T, im impl.Impl, sc model.Scenario, fn func(t *testing.T, r *Rig)) {
	RequireBubbles(t, im)
	synctest.Test(t, func(t *testing.T) {
		rig, err := NewRig(im, &sc)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := rig.Close(); err != nil {
				t.Error(err)
			}
		}()
		fn(t, rig)
	})
}
