package assume

import (
	"testing"

	"github.com/arahe-dev/hertasdk/lab/impl"
	"github.com/arahe-dev/hertasdk/lab/model"
	"github.com/arahe-dev/hertasdk/lab/workload"
)

// held is a script whose first attempt holds until released, then succeeds.
var held = model.Script{{Block: model.BlockHold}}

// calls declares n calls of op with script s and key.
func calls(n, op int, key string, s model.Script) []model.Call {
	out := make([]model.Call, n)
	for i := range out {
		out[i] = model.Call{Op: op, Key: key, Script: s}
	}
	return out
}

// expect fails t unless each listed call is in phase p.
func expect(t *testing.T, r *workload.Rig, p model.Phase, ids ...int) {
	t.Helper()
	v := r.View()
	for _, id := range ids {
		if v.Phases[id] != p {
			t.Fatalf("c%d is %v, want %v (all: %v)", id, v.Phases[id], p, v.Phases)
		}
	}
}

// expectResult fails t unless call id returned with class c.
func expectResult(t *testing.T, r *workload.Rig, id int, c impl.Class) {
	t.Helper()
	expect(t, r, model.Done, id)
	if got := r.Result(id).Class; got != c {
		t.Fatalf("c%d returned %v, want %v", id, got, c)
	}
}

// count is how many calls are in phase p.
func count(r *workload.Rig, p model.Phase) int {
	n := 0
	for _, x := range r.View().Phases {
		if x == p {
			n++
		}
	}
	return n
}

// op1 is a one-resource operation.
func op1(name string, adm impl.Admission, units int64) model.Op {
	return model.Op{Name: name, Effect: impl.Pure, Admission: adm, Resources: []impl.Req{{Name: "r", Units: units}}}
}

// start starts the calls one at a time, settling after each, so arrival
// order is exactly the argument order.
func start(r *workload.Rig, ids ...int) {
	for _, id := range ids {
		r.Start(id)
		r.View()
	}
}

// holders lists the calls currently holding.
func holders(r *workload.Rig) []int {
	var out []int
	for i, p := range r.View().Phases {
		if p == model.Holding {
			out = append(out, i)
		}
	}
	return out
}
