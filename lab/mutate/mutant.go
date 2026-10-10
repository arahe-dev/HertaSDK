package mutate

import (
	"fmt"
	"slices"
	"strings"
)

// Operator names a mutation operator.
type Operator string

const (
	CondNegate   Operator = "cond-negate"   // == ↔ !=, < ↔ >=, > ↔ <=
	CondBoundary Operator = "cond-boundary" // < ↔ <=, > ↔ >=
	CondLogic    Operator = "cond-logic"    // && ↔ ||
	CondNot      Operator = "cond-not"      // !x → x
	StmtDelete   Operator = "stmt-delete"   // drop a call, send, receive, ++/--, assignment or go
	StmtSwap     Operator = "stmt-swap"     // swap two adjacent effect statements
	BoolFlip     Operator = "bool-flip"     // true ↔ false in a return or assignment
	ErrNil       Operator = "err-nil"       // return …, err → return …, nil
	IntOff       Operator = "int-off"       // integer literal ± 1
	DeferDelete  Operator = "defer-delete"  // drop a defer: never release
	DeferInline  Operator = "defer-inline"  // defer f() → f(): release at once
)

// Operators lists every operator in report order.
var Operators = []Operator{CondNegate, CondBoundary, CondLogic, CondNot, StmtDelete,
	StmtSwap, BoolFlip, ErrNil, IntOff, DeferDelete, DeferInline}

// Mutant is one syntactic change to one file.
type Mutant struct {
	ID   int
	File string // slash path relative to the module root
	Func string // enclosing function, e.g. "(*Resource).acquire"
	Line int
	Op   Operator
	From string // the replaced source, whitespace-collapsed
	To   string // its replacement
	Nth  int    // occurrence of this exact change within Func

	edits []edit
}

// edit replaces src[off:end] with text.
type edit struct {
	off, end int
	text     string
}

// Apply returns src with the mutant's edits spliced in.
func (m Mutant) Apply(src []byte) []byte {
	es := slices.Clone(m.edits)
	slices.SortFunc(es, func(a, b edit) int { return b.off - a.off })
	out := slices.Clone(src)
	for _, e := range es {
		out = slices.Concat(out[:e.off], []byte(e.text), out[e.end:])
	}
	return out
}

// Key identifies a mutant independently of line numbers, so a triage
// verdict survives unrelated edits to the file.
func (m Mutant) Key() string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%d", m.File, m.Func, m.Op, m.From, m.To, m.Nth)
}

// Where is file:line.
func (m Mutant) Where() string { return fmt.Sprintf("%s:%d", m.File, m.Line) }

// compact collapses whitespace runs so a snippet fits one CSV cell.
func compact(s string) string { return strings.Join(strings.Fields(s), " ") }
