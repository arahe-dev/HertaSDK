package retrycheck

import (
	"go/ast"
	"go/token"
	"strings"
)

func keyHeader(value string) bool {
	return strings.EqualFold(value, "Idempotency-Key") ||
		strings.EqualFold(value, "X-Idempotency-Key") ||
		strings.EqualFold(value, "PayPal-Request-Id")
}

func (s source) keyValue(n ast.Node) ast.Expr {
	switch n := n.(type) {
	case *ast.CallExpr:
		switch name(n.Fun) {
		case "SetIdempotencyKey", "WithIdempotencyKey":
			if len(n.Args) == 1 {
				return n.Args[0]
			}
		case "Set", "Add":
			if len(n.Args) == 2 && keyHeader(s.stringValue(n.Args[0])) {
				return n.Args[1]
			}
		}
	case *ast.AssignStmt:
		for i, lhs := range n.Lhs {
			if sel, ok := lhs.(*ast.SelectorExpr); ok && sel.Sel.Name == "IdempotencyKey" && i < len(n.Rhs) {
				return n.Rhs[i]
			}
		}
	case *ast.KeyValueExpr:
		if name(n.Key) == "IdempotencyKey" {
			return n.Value
		}
	}
	return nil
}

func (s source) keyMint(expr ast.Expr) bool {
	return anyIn(expr, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return false
		}
		switch name(call.Fun) {
		case "NewIdempotencyKey", "newIdempotencyKey", "GenerateIdempotencyKey", "generateIdempotencyKey":
			return true
		case "New", "NewString", "NewRandom", "NewV4", "GenerateUUID":
			p := s.imported(call.Fun)
			return strings.HasSuffix(p, "/uuid") || p == "github.com/hashicorp/go-uuid" ||
				strings.HasPrefix(p, "github.com/gofrs/uuid/")
		}
		return false
	})
}

func (s source) attemptStart(stack []ast.Node) token.Pos {
	for i := len(stack) - 1; i >= 0; i-- {
		switch n := stack[i].(type) {
		case *ast.ForStmt:
			if i+1 < len(stack) && stack[i+1] == n.Body && retryLoop(n) {
				return n.Body.Pos()
			}
		case *ast.FuncLit:
			if i > 0 {
				if c, ok := stack[i-1].(*ast.CallExpr); ok && s.callback(c, n) {
					return n.Body.Pos()
				}
			}
			return token.NoPos
		case *ast.FuncDecl:
			return token.NoPos
		}
	}
	return token.NoPos
}

func (s source) checkKey(n ast.Node, stack []ast.Node) {
	value := s.keyValue(n)
	if value == nil {
		return
	}
	floor := s.attemptStart(stack)
	if !floor.IsValid() {
		return
	}
	fresh := s.keyMint(value)
	if id, ok := ast.Unparen(value).(*ast.Ident); !fresh && ok {
		fresh = preceding(id.Name, stack, floor, func(_ token.Pos, value ast.Expr) bool {
			return s.keyMint(value)
		})
	}
	if fresh {
		s.report(freshKeyRule, n, "idempotency key is minted inside a retry attempt")
	}
}
