// Package gen is the compiled form of herta/examples/charge.he plus the
// hand-written host code its handler body expects. charge_gen.go is
// produced by `herta build -pkg gen herta/examples/charge.he`; this file is
// the support side: the result type and the business function the body
// calls. It exists to prove the generated code compiles and runs against
// hertasdk.
package gen

import "errors"

// receipt is the op's result type. err is a fake provider error channel so
// the example can exercise both the success and the classified-fail path.
type receipt struct {
	cents uint64
	err   error
}

var errProvider = errors.New("charge provider unavailable")

// do_charge stands in for real business logic: it fails when asked to
// charge nothing, which the handler classifies as transient.
func do_charge(user_id string, cents uint64) receipt {
	if cents == 0 {
		return receipt{err: errProvider}
	}
	return receipt{cents: cents}
}
