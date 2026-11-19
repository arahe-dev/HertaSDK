package lang

import (
	"fmt"
	"strings"
)

// CheckClosed adds closed-world checks without changing ordinary Check.
func CheckClosed(prog *Program) []Diagnostic {
	diags := Check(prog)
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *Op:
			if d.Retry != nil && d.Classify == nil {
				diags = append(diags, Diagnostic{d.Line, d.Col, fmt.Sprintf(
					"HC016 path %s: retry requires declared classification in closed mode; add classify { ... , _ => permanent }", opName(d))})
			}
		case *Handler:
			if constructsOutcome(d) {
				diags = append(diags, Diagnostic{d.Line, d.Col, fmt.Sprintf(
					"HC018 path %s::%s: handler constructs an outcome in closed mode; return the error and declare classify instead", d.Runtime, d.Name)})
			}
		}
	}
	return diags
}

func constructsOutcome(h *Handler) bool {
	var body strings.Builder
	for _, stmt := range h.Body {
		switch s := stmt.(type) {
		case *FailStmt:
			return true
		case *RawStmt:
			if len(s.Fails) > 0 {
				return true
			}
			body.WriteString(s.Text)
			body.WriteByte('\n')
		}
	}
	ts, _ := Lex(body.String())
	for i := 0; i+3 < len(ts); i++ {
		if ts[i].Kind == TokIdent && ts[i].Text == "hertasdk" && ts[i+1].Text == "." &&
			((ts[i+2].Text == "Fail" && ts[i+3].Kind == TokLParen) ||
				(ts[i+2].Text == "Failure" && ts[i+3].Kind == TokLBrace)) {
			return true
		}
	}
	return false
}

// Callees returns the declared and syntactically visible callees, in source
// order without duplicates. Names are qualified within op's runtime.
func Callees(op *Op, h *Handler) []string {
	names := append([]string(nil), op.Calls...)
	if h != nil {
		for _, edge := range handlerEdges(h) {
			names = append(names, edge.name)
		}
	}
	var out []string
	seen := map[string]bool{}
	for _, name := range names {
		name = op.Runtime + "::" + name
		if !seen[name] {
			out = append(out, name)
			seen[name] = true
		}
	}
	return out
}
