package lang

import (
	"fmt"
	"strconv"
)

// Diagnostic is a compile-time contract error.
type Diagnostic struct {
	Line, Col int
	Msg       string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%d:%d: %s", d.Line, d.Col, d.Msg)
}

// Check runs the contract checks of docs/SPEC.md §3 on a parsed program.
// It returns every diagnostic it can find in one pass (parse errors still
// abort earlier, in Parse).
func Check(prog *Program) []Diagnostic {
	var diags []Diagnostic
	at := func(d Diagnostic) { diags = append(diags, d) }

	rts := map[string]*Runtime{}
	ops := map[string]*Op{} // "runtime::name"

	for _, d := range prog.Decls {
		switch d := d.(type) {
		case *Runtime:
			if _, dup := rts[d.Name]; dup {
				at(Diagnostic{d.Line, d.Col, fmt.Sprintf("runtime %q declared twice", d.Name)})
				continue
			}
			rts[d.Name] = d
		case *Op:
			key := d.Runtime + "::" + d.Name
			if _, dup := ops[key]; dup {
				at(Diagnostic{d.Line, d.Col, fmt.Sprintf("op %q declared twice", key)})
				continue
			}
			ops[key] = d
		}
	}

	for _, d := range prog.Decls {
		switch d := d.(type) {
		case *Runtime:
			checkRuntime(d, at)
		case *Op:
			checkOp(d, rts[d.Runtime], at)
		case *Handler:
			checkHandler(d, ops[d.Runtime+"::"+d.Name], at)
		}
	}
	return diags
}

func checkRuntime(rt *Runtime, at func(Diagnostic)) {
	if len(rt.Resources) == 0 {
		at(Diagnostic{rt.Line, rt.Col, fmt.Sprintf("runtime %q declares no resources", rt.Name)})
	}
	seen := map[string]bool{}
	for _, r := range rt.Resources {
		if seen[r.Name] {
			at(Diagnostic{r.Line, r.Col, fmt.Sprintf("duplicate resource %q in runtime %q", r.Name, rt.Name)})
			continue
		}
		seen[r.Name] = true
		if r.Capacity <= 0 {
			at(Diagnostic{r.Line, r.Col, fmt.Sprintf("resource %q: capacity must be > 0", r.Name)})
		}
	}
}

func checkOp(op *Op, rt *Runtime, at func(Diagnostic)) {
	if rt == nil {
		at(Diagnostic{op.Line, op.Col, fmt.Sprintf("op %q: runtime %q is not declared", op.Name, op.Runtime)})
		return
	}
	// Retry-policy legality is checked before unsafe-retry, matching
	// hertasdk's deterministic order (contract §3.2).
	if op.Effect == "" {
		at(Diagnostic{op.Line, op.Col, fmt.Sprintf("op %q: effect is required (pure|idempotent|nonidempotent)", op.Name)})
	}
	if rt != nil {
		declared := map[string]int64{}
		for _, r := range rt.Resources {
			declared[r.Name] = r.Capacity
		}
		seen := map[string]bool{}
		for _, r := range op.Resources {
			cap_, ok := declared[r.Name]
			if !ok {
				at(Diagnostic{r.Line, r.Col, fmt.Sprintf("op %q: resource %q not declared on runtime %q", op.Name, r.Name, rt.Name)})
				continue
			}
			if r.Units <= 0 {
				at(Diagnostic{r.Line, r.Col, fmt.Sprintf("op %q: units for %q must be > 0", op.Name, r.Name)})
			} else if r.Units > cap_ {
				at(Diagnostic{r.Line, r.Col, fmt.Sprintf("op %q: units %d for %q exceed capacity %d", op.Name, r.Units, r.Name, cap_)})
			}
			if seen[r.Name] {
				at(Diagnostic{r.Line, r.Col, fmt.Sprintf("op %q: resource %q listed twice", op.Name, r.Name)})
			}
			seen[r.Name] = true
		}
	}
	if op.Key != "" {
		found := false
		for _, prm := range op.Params {
			if prm.Name == op.Key {
				found = true
				if prm.Type != "string" {
					at(Diagnostic{op.Line, op.Col, fmt.Sprintf("op %q: serialize key %q must be a string parameter, got %s", op.Name, op.Key, prm.Type)})
				}
			}
		}
		if !found {
			at(Diagnostic{op.Line, op.Col, fmt.Sprintf("op %q: serialize key %q is not a parameter", op.Name, op.Key)})
		}
	}
	if op.Timeout != "" {
		i := 0
		for i < len(op.Timeout) && op.Timeout[i] >= '0' && op.Timeout[i] <= '9' {
			i++
		}
		if n, err := strconv.ParseInt(op.Timeout[:i], 10, 64); err != nil || n <= 0 {
			at(Diagnostic{op.Line, op.Col, fmt.Sprintf("op %q: timeout must be > 0", op.Name)})
		}
	}
	if op.Retry != nil {
		for _, o := range op.Retry.On {
			switch o.Name {
			case "success", "permanent", "throttled":
				at(Diagnostic{o.Line, o.Col, fmt.Sprintf("op %q: retry on %s is never allowed", op.Name, o.Name)})
			}
		}
		if op.Retry.Max < 1 {
			at(Diagnostic{op.Retry.Line, op.Retry.Col, fmt.Sprintf("op %q: retry max must be >= 1", op.Name)})
		}
		// ErrUnsafeRetry analog, reported after invalid-policy errors.
		if op.Effect == "nonidempotent" {
			for _, o := range op.Retry.On {
				if o.Name == "uncertain" {
					at(Diagnostic{o.Line, o.Col, fmt.Sprintf("op %q: retrying uncertain outcomes is unsafe for effect nonidempotent", op.Name)})
				}
			}
		}
	}
}

func checkHandler(h *Handler, op *Op, at func(Diagnostic)) {
	if op == nil {
		at(Diagnostic{h.Line, h.Col, fmt.Sprintf("handler %q: no matching op declared", h.Runtime+"::"+h.Name)})
		return
	}
	if len(h.Params) != len(op.Params) {
		at(Diagnostic{h.Line, h.Col, fmt.Sprintf("handler %q: %d params, op declares %d", h.Name, len(h.Params), len(op.Params))})
	} else {
		for i, prm := range h.Params {
			if prm.Name != op.Params[i].Name || prm.Type != op.Params[i].Type {
				at(Diagnostic{h.Line, h.Col, fmt.Sprintf("handler %q: param %d is %s:%s, op declares %s:%s", h.Name, i, prm.Name, prm.Type, op.Params[i].Name, op.Params[i].Type)})
			}
		}
	}
	want := append([]string{op.Result}, "error")
	if len(h.Results) != 2 || h.Results[0] != want[0] || h.Results[1] != "error" {
		at(Diagnostic{h.Line, h.Col, fmt.Sprintf("handler %q: results must be (%s, error)", h.Name, op.Result)})
	}
	for _, s := range h.Body {
		if f, ok := s.(*FailStmt); ok {
			switch f.Outcome {
			case "transient", "uncertain", "permanent", "throttled":
			case "success":
				at(Diagnostic{f.Line, f.Col, fmt.Sprintf("handler %q: cannot fail with outcome success", h.Name)})
			default:
				at(Diagnostic{f.Line, f.Col, fmt.Sprintf("handler %q: invalid fail outcome %q", h.Name, f.Outcome)})
			}
		}
	}
}
