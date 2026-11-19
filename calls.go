package hertasdk

import "fmt"

type callFrameKey struct{}

type callFrame struct {
	name   string
	key    string
	rt     *Runtime
	closed bool
	calls  map[string]bool
	parent *callFrame
}

func callViolation(kind error, parent, callee string) error {
	return &Failure{Outcome: Permanent, Err: fmt.Errorf("%w: %s -> %s", kind, parent, callee)}
}
