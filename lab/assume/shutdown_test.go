package assume

import (
	"testing"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// TestShutdownPhases
//
// Assumption: Shutdown runs three phases in order (§6): stop admitting;
// wake every waiter, key or resource, with ErrRuntimeStopping; drain the
// calls already executing, never cancelling them; return nil only when
// all admitted calls have finished.
//
// Why it matters: this is Mirai's SIGTERM path (HTTP drain, then
// rt.Shutdown(15s), then the DB pool). Work that was admitted must finish;
// work that was only queued must fail fast so the drain can end.
//
// Verdict: with one call executing on the budget, one queued on it, one
// executing under key "k" and one queued on "k": Shutdown withdraws both
// waiters at once, refuses a new call (counted in Rejected), stays
// pending until both executing calls are released, then returns nil with
// Admitted = Finished. A second Shutdown also returns nil.
func TestShutdownPhases(t *testing.T) {
	workload.Each(t, func(t *testing.T, im impl.Impl) {
		keyed := model.Op{Name: "keyed", Effect: impl.Idempotent, Keyed: true}
		sc := model.Scenario{
			Specs: []impl.Spec{{Name: "r", Capacity: 1}},
			Ops:   []model.Op{op1("budget", impl.Wait, 1), keyed},
			Calls: []model.Call{
				{Op: 0, Script: held}, {Op: 0, Script: held},
				{Op: 1, Key: "k", Script: held}, {Op: 1, Key: "k", Script: held},
				{Op: 0, Script: held},
			},
		}
		workload.Bubble(t, im, sc, func(t *testing.T, r *workload.Rig) {
			start(r, 0, 1, 2, 3)
			expect(t, r, model.Holding, 0, 2)
			expect(t, r, model.Parked, 1, 3)

			r.Shutdown(false)
			expectResult(t, r, 1, impl.ClassStopping)
			expectResult(t, r, 3, impl.ClassStopping)
			expect(t, r, model.Holding, 0, 2)
			start(r, 4)
			expectResult(t, r, 4, impl.ClassStopping)

			r.Release(0)
			if v := r.View(); v.Shutdowns[0].Returned {
				t.Fatal("Shutdown returned while c2 was still executing")
			}
			r.Release(2)
			r.Shutdown(false)
			v := r.View()
			for i, sd := range v.Shutdowns {
				if !sd.Returned || sd.Class != impl.ClassNil {
					t.Errorf("Shutdown #%d: %+v, want returned nil", i, sd)
				}
			}
			want := impl.Stats{Admitted: 4, Rejected: 1, Started: 2, Finished: 4}
			if v.Stats != want {
				t.Errorf("stats %+v, want %+v", v.Stats, want)
			}
		})
	})
}
