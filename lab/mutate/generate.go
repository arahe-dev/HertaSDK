package mutate

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"strconv"
)

// Generate enumerates every distinct mutant of one Go source file. name is
// the file's slash path relative to the module root; it is recorded, not
// opened. Mutants that produce the same token stream as the original or
// an earlier mutant (say, deleting either of two identical calls) are
// dropped.
func Generate(name string, src []byte) ([]Mutant, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	g := &gen{
		tf: fset.File(f.Pos()), src: src, file: name,
		nth: map[string]int{}, seen: map[[32]byte]bool{fingerprint(src): true},
	}
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			g.stack = g.stack[:len(g.stack)-1]
			return true
		}
		if d, ok := n.(*ast.GenDecl); ok && d.Tok == token.IMPORT {
			return false
		}
		g.stack = append(g.stack, n)
		g.visit(n)
		return true
	})
	return g.out, nil
}

type gen struct {
	tf    *token.File
	src   []byte
	file  string
	stack []ast.Node // ancestors of the node being visited, itself included
	out   []Mutant
	nth   map[string]int    // occurrences per Key without Nth
	seen  map[[32]byte]bool // token streams already emitted
}

func (g *gen) visit(n ast.Node) {
	switch n := n.(type) {
	case *ast.BinaryExpr:
		g.binary(n)
	case *ast.UnaryExpr:
		if n.Op == token.NOT {
			g.emit(CondNot, g.splice(n.OpPos, n.X.Pos(), ""))
		}
	case *ast.BlockStmt:
		g.stmts(n.List)
	case *ast.CaseClause:
		g.stmts(n.Body)
	case *ast.CommClause:
		g.stmts(n.Body)
	case *ast.ReturnStmt:
		g.flipBools(n.Results)
		g.errNil(n)
	case *ast.AssignStmt:
		g.flipBools(n.Rhs)
	case *ast.KeyValueExpr:
		g.flipBools([]ast.Expr{n.Value})
	case *ast.BasicLit:
		g.intOff(n)
	}
}

var (
	negated  = map[token.Token]token.Token{token.EQL: token.NEQ, token.NEQ: token.EQL, token.LSS: token.GEQ, token.GEQ: token.LSS, token.GTR: token.LEQ, token.LEQ: token.GTR}
	boundary = map[token.Token]token.Token{token.LSS: token.LEQ, token.LEQ: token.LSS, token.GTR: token.GEQ, token.GEQ: token.GTR}
	logic    = map[token.Token]token.Token{token.LAND: token.LOR, token.LOR: token.LAND}
)

func (g *gen) binary(n *ast.BinaryExpr) {
	end := n.OpPos + token.Pos(len(n.Op.String()))
	for _, r := range []struct {
		op  Operator
		alt map[token.Token]token.Token
	}{{CondNegate, negated}, {CondBoundary, boundary}, {CondLogic, logic}} {
		if to, ok := r.alt[n.Op]; ok {
			g.emit(r.op, g.splice(n.OpPos, end, to.String()))
		}
	}
}

// stmts mutates one statement list: deletions, defer removal and
// inlining, and swaps of adjacent effect statements (where acquire and
// release order lives).
func (g *gen) stmts(list []ast.Stmt) {
	for i, s := range list {
		switch s := s.(type) {
		case *ast.DeferStmt:
			g.emit(DeferDelete, g.splice(s.Pos(), s.End(), ""))
			g.emit(DeferInline, g.splice(s.Defer, s.Call.Pos(), ""))
		default:
			if effect(s) {
				g.emit(StmtDelete, g.splice(s.Pos(), s.End(), ""))
			}
		}
		if i > 0 && swappable(list[i-1], s) {
			a, b := list[i-1], s
			g.emit(StmtSwap,
				g.splice(a.Pos(), a.End(), g.text(b.Pos(), b.End())),
				g.splice(b.Pos(), b.End(), g.text(a.Pos(), a.End())))
		}
	}
}

// effect reports a statement whose only job is a side effect, so deleting
// or reordering it still compiles in most cases.
func effect(s ast.Stmt) bool {
	switch s := s.(type) {
	case *ast.ExprStmt, *ast.SendStmt, *ast.IncDecStmt, *ast.GoStmt:
		return true
	case *ast.AssignStmt:
		return s.Tok != token.DEFINE
	}
	return false
}

// swappable pairs two effect statements, or two defers (whose LIFO order
// is release order). A defer and a plain statement are not paired: the
// defer runs at return either way.
func swappable(a, b ast.Stmt) bool {
	_, da := a.(*ast.DeferStmt)
	_, db := b.(*ast.DeferStmt)
	return da && db || effect(a) && effect(b)
}

func (g *gen) flipBools(es []ast.Expr) {
	for _, e := range es {
		if id, ok := e.(*ast.Ident); ok && (id.Name == "true" || id.Name == "false") {
			to := map[string]string{"true": "false", "false": "true"}[id.Name]
			g.emit(BoolFlip, g.splice(id.Pos(), id.End(), to))
		}
	}
}

// errNil makes a return in a function whose last result is error report
// nil there instead.
func (g *gen) errNil(r *ast.ReturnStmt) {
	ft := g.funcType()
	if ft == nil || ft.Results == nil || len(r.Results) == 0 {
		return
	}
	fields := ft.Results.List
	if id, ok := fields[len(fields)-1].Type.(*ast.Ident); !ok || id.Name != "error" {
		return
	}
	if len(r.Results) != ft.Results.NumFields() {
		return // return f() forwarding several results
	}
	last := r.Results[len(r.Results)-1]
	if id, ok := last.(*ast.Ident); ok && id.Name == "nil" {
		return
	}
	g.emit(ErrNil, g.splice(last.Pos(), last.End(), "nil"))
}

func (g *gen) intOff(l *ast.BasicLit) {
	if l.Kind != token.INT {
		return
	}
	v, err := strconv.ParseInt(l.Value, 0, 64)
	if err != nil {
		return
	}
	for _, d := range []int64{1, -1} {
		g.emit(IntOff, g.splice(l.Pos(), l.End(), strconv.FormatInt(v+d, 10)))
	}
}

// funcType is the innermost enclosing function's signature.
func (g *gen) funcType() *ast.FuncType {
	for i := len(g.stack) - 1; i >= 0; i-- {
		switch f := g.stack[i].(type) {
		case *ast.FuncLit:
			return f.Type
		case *ast.FuncDecl:
			return f.Type
		}
	}
	return nil
}

// funcName names the enclosing declaration, e.g. "(*Resource).acquire".
func (g *gen) funcName() string {
	for i := len(g.stack) - 1; i >= 0; i-- {
		f, ok := g.stack[i].(*ast.FuncDecl)
		if !ok {
			continue
		}
		if f.Recv == nil || len(f.Recv.List) == 0 {
			return f.Name.Name
		}
		return "(" + recvName(f.Recv.List[0].Type) + ")." + f.Name.Name
	}
	return ""
}

func recvName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return "*" + recvName(t.X)
	case *ast.IndexExpr:
		return recvName(t.X)
	case *ast.IndexListExpr:
		return recvName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return "?"
}

func (g *gen) splice(from, to token.Pos, text string) edit {
	return edit{off: g.tf.Offset(from), end: g.tf.Offset(to), text: text}
}

func (g *gen) text(from, to token.Pos) string {
	return string(g.src[g.tf.Offset(from):g.tf.Offset(to)])
}

// emit records a mutant unless it reproduces the original or an earlier
// mutant token for token.
func (g *gen) emit(op Operator, es ...edit) {
	m := Mutant{File: g.file, Func: g.funcName(), Op: op, edits: es,
		Line: g.tf.Line(g.tf.Pos(es[0].off))}
	sum := fingerprint(m.Apply(g.src))
	if g.seen[sum] {
		return
	}
	g.seen[sum] = true
	m.From, m.To = snippet(g.src, es)
	k := m.Key()
	m.Nth = g.nth[k]
	g.nth[k]++
	g.out = append(g.out, m)
}

// snippet renders what changed: the first edit's text, or for a swap the
// two statements in old and new order.
func snippet(src []byte, es []edit) (from, to string) {
	from = clip(compact(string(src[es[0].off:es[0].end])))
	to = clip(compact(es[0].text))
	if len(es) == 2 {
		to = from + " ⇄ " + clip(compact(es[0].text))
		from = "swap"
	}
	return from, to
}

func clip(s string) string {
	if r := []rune(s); len(r) > 80 {
		return string(r[:79]) + "…"
	}
	return s
}

// fingerprint hashes src's token stream, so mutants that differ only in
// whitespace or comments count as one.
func fingerprint(src []byte) [32]byte {
	fset := token.NewFileSet()
	var s scanner.Scanner
	s.Init(fset.AddFile("", -1, len(src)), src, nil, 0)
	h := sha256.New()
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.SEMICOLON {
			lit = ""
		}
		fmt.Fprintf(h, "%d %s\x00", tok, lit)
	}
	return [32]byte(h.Sum(nil))
}
