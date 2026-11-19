package gogen

import (
	"fmt"
	"strings"

	"github.com/arahe-dev/hertasdk/herta/lang"
)

// planCall is one `call name ( args )` in token-joined body text:
// text[start:end] is the whole call.
type planCall struct {
	name       string
	args       []string
	start, end int
}

// findCalls scans token-joined statement text for `call <ident> (`; it
// mirrors the checker's explicit edges (lang.handlerEdges), skipping string
// and char literals.
func findCalls(text string) []planCall {
	var out []planCall
	for i := 0; i < len(text); i++ {
		if c := text[i]; c == '"' || c == '\'' || c == '`' {
			i = skipLit(text, i)
			continue
		}
		if !strings.HasPrefix(text[i:], "call ") || (i > 0 && isIdent(text[i-1])) {
			continue
		}
		j := i + len("call ")
		k := j
		for k < len(text) && isIdent(text[k]) {
			k++
		}
		if k == j || !strings.HasPrefix(text[k:], " (") {
			continue
		}
		args, end, ok := splitArgs(text, k+2)
		if !ok {
			continue
		}
		out = append(out, planCall{name: text[j:k], args: args, start: i, end: end})
		i = end - 1
	}
	return out
}

func isIdent(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// skipLit returns the index of the closing quote of the literal at i.
func skipLit(text string, i int) int {
	q := text[i]
	j := i + 1
	for j < len(text) && text[j] != q {
		if text[j] == '\\' && q != '`' {
			j++
		}
		j++
	}
	return j
}

// splitArgs reads arguments from just after `(` to the matching `)`,
// split at top-level commas. end is just past the `)`.
func splitArgs(text string, i int) (args []string, end int, ok bool) {
	depth, start := 0, i
	for ; i < len(text); i++ {
		switch c := text[i]; c {
		case '"', '\'', '`':
			i = skipLit(text, i)
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth == 0 {
				if a := strings.TrimSpace(text[start:i]); a != "" || len(args) > 0 {
					args = append(args, a)
				}
				return args, i + 1, c == ')'
			}
			depth--
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(text[start:i]))
				start = i + 1
			}
		}
	}
	return nil, 0, false
}

// lowerCalls executes each `call` through the plan's operation: the same
// edge the checker and the closed Calls policy saw. A statement that is
// only a call propagates the callee's error unchanged (tag included).
func (g opGen) lowerCalls(text, result string) (string, error) {
	calls := findCalls(text)
	var b strings.Builder
	last := 0
	for _, c := range calls {
		callee := g.ops[g.op.Runtime+"::"+c.name]
		if callee == nil {
			return "", fmt.Errorf("op %s::%s calls unknown op %s", g.op.Runtime, g.op.Name, c.name)
		}
		if len(c.args) != len(callee.Params) {
			return "", fmt.Errorf("op %s::%s calls %s with %d arguments, want %d",
				g.op.Runtime, g.op.Name, c.name, len(c.args), len(callee.Params))
		}
		base := export(callee.Runtime) + export(callee.Name)
		fields := make([]string, len(c.args))
		for i, a := range c.args {
			fields[i] = export(callee.Params[i].Name) + ": " + a
		}
		do := fmt.Sprintf("p.%s.Do(ctx, %sInput{%s})", export(callee.Name), base, strings.Join(fields, ", "))
		if c.start == 0 && c.end == len(text) {
			return fmt.Sprintf("if _, err := %s; err != nil { return *new(%s), err }", do, result), nil
		}
		b.WriteString(text[last:c.start])
		b.WriteString(do)
		last = c.end
	}
	b.WriteString(text[last:])
	return b.String(), nil
}

func planName(runtime string) string { return export(runtime) + "Plan" }

// genPlan emits the runtime's executable call plan: one field per op and a
// constructor binding each handler to it.
//
//	type MainPlan struct{ Charge *hertasdk.Operation[MainChargeInput, uint64]; ... }
//	func NewMainPlan(rt *hertasdk.Runtime) (*MainPlan, error)
func genPlan(b *strings.Builder, runtime string, order []*lang.Op) {
	name := planName(runtime)
	fmt.Fprintf(b, "// %s holds runtime %q's operations; `call` statements execute through it.\n", name, runtime)
	fmt.Fprintf(b, "type %s struct {\n", name)
	for _, op := range order {
		if op.Runtime == runtime {
			base := export(op.Runtime) + export(op.Name)
			fmt.Fprintf(b, "\t%s *hertasdk.Operation[%sInput, %s]\n", export(op.Name), base, goType(op.Result))
		}
	}
	b.WriteString("}\n\n")
	fmt.Fprintf(b, "// New%s constructs every operation of runtime %q on rt.\n", name, runtime)
	fmt.Fprintf(b, "func New%s(rt *hertasdk.Runtime) (*%s, error) {\n\tp := &%s{}\n\tvar err error\n", name, name, name)
	for _, op := range order {
		if op.Runtime == runtime {
			fmt.Fprintf(b, "\tif p.%s, err = new%s%s(rt, p); err != nil {\n\t\treturn nil, err\n\t}\n",
				export(op.Name), export(op.Runtime), export(op.Name))
		}
	}
	b.WriteString("\treturn p, nil\n}\n\n")
}

// trackDispatch closes `evidence dispatch` over the call graph: an op that
// can reach a tracked op tracks too, so a nested dispatch reaches every
// enclosing attempt (hertasdk.Dispatch) without per-caller annotation.
func trackDispatch(order []*lang.Op, handlers map[string]*lang.Handler) map[string]bool {
	tracks := map[string]bool{}
	edges := map[string][]string{}
	for _, op := range order {
		name := op.Runtime + "::" + op.Name
		tracks[name] = op.TrackDispatch
		edges[name] = lang.Callees(op, handlers[name])
	}
	for changed := true; changed; {
		changed = false
		for name, callees := range edges {
			for _, c := range callees {
				if tracks[c] && !tracks[name] {
					tracks[name], changed = true, true
				}
			}
		}
	}
	return tracks
}
