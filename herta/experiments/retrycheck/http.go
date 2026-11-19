package retrycheck

import (
	"go/ast"
	"go/token"
)

const retryableHTTP = "github.com/hashicorp/go-retryablehttp"

func (s source) defaultClient(expr ast.Expr) bool {
	switch n := ast.Unparen(expr).(type) {
	case *ast.SelectorExpr:
		return name(n) == "DefaultClient" && s.imported(n) == retryableHTTP
	case *ast.CallExpr:
		if name(n.Fun) == "NewClient" && s.imported(n.Fun) == retryableHTTP && len(n.Args) == 0 {
			return true
		}
		if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "StandardClient" {
			return s.defaultClient(sel.X)
		}
	}
	return false
}

func (s source) newWriteRequest(expr ast.Expr) bool {
	c, ok := ast.Unparen(expr).(*ast.CallExpr)
	if !ok || (s.imported(c.Fun) != "net/http" && s.imported(c.Fun) != retryableHTTP) {
		return false
	}
	i := 0
	switch name(c.Fun) {
	case "NewRequest":
	case "NewRequestWithContext":
		i = 1
	default:
		return false
	}
	if len(c.Args) <= i {
		return false
	}
	method := s.stringValue(c.Args[i])
	return method == "POST" || method == "PATCH"
}

func (s source) checkRetryClient(c *ast.CallExpr, stack []ast.Node) {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	switch sel.Sel.Name {
	case "Post", "PostForm", "Do":
	default:
		return
	}
	matches := s.imported(sel) == retryableHTTP || s.defaultClient(sel.X)
	if id, ok := ast.Unparen(sel.X).(*ast.Ident); !matches && ok {
		matches = preceding(id.Name, stack, token.NoPos, func(_ token.Pos, v ast.Expr) bool {
			return s.defaultClient(v)
		})
	}
	if !matches {
		return
	}
	if sel.Sel.Name == "Do" {
		if len(c.Args) != 1 {
			return
		}
		request, ok := ast.Unparen(c.Args[0]).(*ast.Ident)
		if !ok || !preceding(request.Name, stack, token.NoPos, func(pos token.Pos, v ast.Expr) bool {
			return s.newWriteRequest(v) && !s.keyedBefore(request.Name, pos, c.Pos(), stack)
		}) {
			return
		}
	}
	s.report(retryHTTPRule, c, "default retryable HTTP client sends POST/PATCH with no visible idempotency key")
}

func (s source) keyedBefore(req string, start, end token.Pos, stack []ast.Node) bool {
	for _, n := range stack {
		block, ok := n.(*ast.BlockStmt)
		if !ok {
			continue
		}
		for _, stmt := range block.List {
			if stmt.Pos() <= start || stmt.End() > end {
				continue
			}
			expr, ok := stmt.(*ast.ExprStmt)
			if !ok {
				continue
			}
			c, ok := expr.X.(*ast.CallExpr)
			if !ok || s.keyValue(c) == nil {
				continue
			}
			if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
				if header, ok := sel.X.(*ast.SelectorExpr); ok && header.Sel.Name == "Header" && name(header.X) == req {
					return true
				}
			}
		}
	}
	return false
}

func (s source) checkReplay(fn *ast.FuncDecl) {
	if fn.Recv == nil || (fn.Name.Name != "Do" && fn.Name.Name != "RoundTrip") {
		return
	}
	acceptsRequest := false
	for _, p := range fn.Type.Params.List {
		if ptr, ok := p.Type.(*ast.StarExpr); ok && name(ptr.X) == "Request" {
			acceptsRequest = s.imported(ptr.X) == "net/http" || s.isRequestWrapper(ptr.X)
			if acceptsRequest {
				break
			}
		}
	}
	if !acceptsRequest || anyIn(fn.Body, s.methodGate) {
		return
	}
	if anyIn(fn.Body, func(n ast.Node) bool {
		loop, ok := n.(*ast.ForStmt)
		return ok && retryLoop(loop) && anyIn(loop.Body, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			return ok && (name(c.Fun) == "Do" || name(c.Fun) == "RoundTrip")
		})
	}) {
		s.report(retryHTTPRule, fn.Name, "HTTP retry loop has no visible method or idempotency gate")
	}
}

func (s source) isRequestWrapper(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	for _, file := range s.pass.Files {
		fileSource := source{imports: importPaths(file)}
		for _, decl := range file.Decls {
			d, ok := decl.(*ast.GenDecl)
			if !ok || d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				t := spec.(*ast.TypeSpec)
				st, ok := t.Type.(*ast.StructType)
				if !ok || t.Name.Name != id.Name {
					continue
				}
				for _, f := range st.Fields.List {
					if ptr, ok := f.Type.(*ast.StarExpr); ok && len(f.Names) == 0 &&
						name(ptr.X) == "Request" && fileSource.imported(ptr.X) == "net/http" {
						return true
					}
				}
			}
		}
	}
	return false
}

func (s source) methodGate(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.IfStmt:
		return s.gateExpr(n.Cond)
	case *ast.SwitchStmt:
		if n.Tag != nil {
			return s.gateExpr(n.Tag)
		}
		for _, stmt := range n.Body.List {
			for _, expr := range stmt.(*ast.CaseClause).List {
				if s.gateExpr(expr) {
					return true
				}
			}
		}
	}
	return false
}

func (s source) gateExpr(expr ast.Expr) bool {
	return anyIn(expr, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			return n.Sel.Name == "Method" || n.Sel.Name == "IdempotencyKey"
		case *ast.CallExpr:
			return name(n.Fun) == "Get" && len(n.Args) == 1 && keyHeader(s.stringValue(n.Args[0]))
		}
		return false
	})
}
