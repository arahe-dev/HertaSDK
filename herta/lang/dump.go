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
		case *ErrorDecl:
			fmt.Fprintf(&b, "error %s ambiguous\n", d.Name)
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
			if len(d.Calls) > 0 {
				fmt.Fprintf(&b, "  calls %s\n", strings.Join(d.Calls, ", "))
			}
			for _, l := range d.Locks {
				fmt.Fprintf(&b, "  lock %s", l.Name)
				if l.Shared {
					b.WriteString(" shared")
				}
				b.WriteByte('\n')
			}
			if d.Dedupe != "" {
				fmt.Fprintf(&b, "  dedupe key %s\n", d.Dedupe)
			}
			if d.Detached {
				b.WriteString("  detached")
				if d.Owner != "" {
					fmt.Fprintf(&b, " for %s", d.Owner)
				}
				b.WriteByte('\n')
			}
			if d.Budget != 0 {
				fmt.Fprintf(&b, "  budget %d\n", d.Budget)
			}
			if d.Classify != nil {
				b.WriteString("  classify {")
				for i, a := range d.Classify {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(&b, "%s => %s", a.Error, a.Outcome)
				}
				b.WriteString("}\n")
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
