package lang

import (
	"fmt"
	"strings"
)

// Dump renders a program's AST in a stable text form used by golden tests
// and `herta ast`.
func Dump(prog *Program) string {
	var b strings.Builder
	for _, d := range prog.Decls {
		switch d := d.(type) {
		case *Runtime:
			fmt.Fprintf(&b, "runtime %s\n", d.Name)
			for _, r := range d.Resources {
				fmt.Fprintf(&b, "  resource %s capacity %d\n", r.Name, r.Capacity)
			}
		case *Op:
			fmt.Fprintf(&b, "op %s::%s(", d.Runtime, d.Name)
			for i, p := range d.Params {
				if i > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(&b, "%s: %s", p.Name, p.Type)
			}
			fmt.Fprintf(&b, ") -> %s\n", d.Result)
			if d.Effect != "" {
				fmt.Fprintf(&b, "  effect %s\n", d.Effect)
			}
			if d.Admission != "" {
				fmt.Fprintf(&b, "  admission %s\n", d.Admission)
			}
			for _, r := range d.Resources {
				fmt.Fprintf(&b, "  resource %s units %d\n", r.Name, r.Units)
			}
			if d.Timeout != "" {
				fmt.Fprintf(&b, "  timeout %s\n", d.Timeout)
			}
			if d.Key != "" {
				fmt.Fprintf(&b, "  serialize key %s\n", d.Key)
			}
			if d.Retry != nil {
				var on []string
				for _, o := range d.Retry.On {
					on = append(on, o.Name)
				}
				fmt.Fprintf(&b, "  retry max %d on {%s}\n", d.Retry.Max, strings.Join(on, ", "))
			}
		case *Handler:
			fmt.Fprintf(&b, "handler %s::%s -> (%s)\n", d.Runtime, d.Name, strings.Join(d.Results, ", "))
			for _, s := range d.Body {
				switch s := s.(type) {
				case *FailStmt:
					fmt.Fprintf(&b, "  fail %s, %s\n", s.Outcome, s.Arg)
				case *RawStmt:
					fmt.Fprintf(&b, "  stmt %s\n", s.Text)
				}
			}
		}
	}
	return b.String()
}
