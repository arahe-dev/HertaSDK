// Package retrycheck provides experimental, syntax-only retry diagnostics.
package retrycheck

import (
	"go/ast"
	"go/constant"
	"go/token"
	"path"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const (
	freshKeyRule  = "RK001"
	retryHTTPRule = "RK002"
	webhookRule   = "RK003"
)

var Analyzer = &analysis.Analyzer{
	Name: "retrycheck",
	Doc:  "find candidate retry bugs: fresh keys, unguarded HTTP replay, and webhook effects without visible dedupe",
	Run:  run,
}

type source struct {
	pass        *analysis.Pass
	imports     map[string]string
	webhookFile bool
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}
		s := source{
			pass:        pass,
			imports:     importPaths(file),
			webhookFile: strings.Contains(strings.ToLower(path.Base(pass.Fset.PositionFor(file.Pos(), false).Filename)), "webhook"),
		}

		ast.PreorderStack(file, nil, func(n ast.Node, stack []ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				s.checkRetryClient(n, stack)
				s.checkKey(n, stack)
			case *ast.AssignStmt, *ast.KeyValueExpr:
				s.checkKey(n, stack)
			case *ast.FuncDecl:
				if n.Body != nil {
					s.checkReplay(n)
					s.checkWebhook(n.Name.Name, n.Type, n.Body, stack)
				}
			case *ast.FuncLit:
				s.checkWebhook("", n.Type, n.Body, stack)
			}
			return true
		})
	}
	return nil, nil
}

func importPaths(file *ast.File) map[string]string {
	paths := make(map[string]string)
	for _, imp := range file.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		alias := path.Base(p)
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			if len(alias) > 1 && alias[0] == 'v' {
				if _, err := strconv.Atoi(alias[1:]); err == nil {
					alias = path.Base(path.Dir(p))
				}
			}
			switch p {
			case retryableHTTP:
				alias = "retryablehttp"
			case "github.com/hashicorp/go-uuid":
				alias = "uuid"
			default:
				if p == "github.com/sethvargo/go-retry" || p == "github.com/avast/retry-go" ||
					strings.HasPrefix(p, "github.com/avast/retry-go/") {
					alias = "retry"
				}
			}
		}
		paths[alias] = p
	}
	return paths
}

func (s source) report(rule string, at ast.Node, message string) {
	s.pass.Report(analysis.Diagnostic{Pos: at.Pos(), End: at.End(), Category: rule, Message: message})
}

func name(n ast.Node) string {
	switch n := n.(type) {
	case *ast.Ident:
		return n.Name
	case *ast.SelectorExpr:
		return n.Sel.Name
	}
	return ""
}

func (s source) imported(n ast.Node) string {
	if n, ok := n.(*ast.SelectorExpr); ok {
		if x, ok := n.X.(*ast.Ident); ok {
			return s.imports[x.Name]
		}
	}
	return ""
}

func (s source) stringValue(n ast.Expr) string {
	switch n := n.(type) {
	case *ast.BasicLit:
		if n.Kind == token.STRING {
			v, _ := strconv.Unquote(n.Value)
			return v
		}
	case *ast.SelectorExpr:
		if s.imported(n) == "net/http" {
			switch n.Sel.Name {
			case "MethodPost":
				return "POST"
			case "MethodPatch":
				return "PATCH"
			}
		}
	}
	if s.pass.TypesInfo != nil {
		if v := s.pass.TypesInfo.Types[n].Value; v != nil && v.Kind() == constant.String {
			return constant.StringVal(v)
		}
	}
	return ""
}

// anyIn does not attribute calls in a nested function to its enclosing body.
func anyIn(root ast.Node, match func(ast.Node) bool) bool {
	found := false
	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil || found {
			return false
		}
		if _, ok := n.(*ast.FuncLit); ok && n != root {
			return false
		}
		found = match(n)
		return !found
	})
	return found
}

// preceding finds the nearest syntactic assignment to an identifier in an
// enclosing block. It deliberately does not follow aliases or branches.
func preceding(id string, stack []ast.Node, floor token.Pos, visit func(token.Pos, ast.Expr) bool) bool {
	for i := len(stack) - 1; i > 0; i-- {
		switch n := stack[i].(type) {
		case *ast.FuncLit, *ast.FuncDecl:
			return false
		case *ast.BlockStmt:
			if n.Pos() < floor {
				return false
			}
			if i+1 == len(stack) {
				continue
			}
			for j := len(n.List) - 1; j >= 0; j-- {
				stmt := n.List[j]
				if stmt.Pos() >= stack[i+1].Pos() {
					continue
				}
				a, ok := stmt.(*ast.AssignStmt)
				if !ok {
					continue
				}
				for k, lhs := range a.Lhs {
					if x, ok := lhs.(*ast.Ident); ok && x.Name == id {
						return k < len(a.Rhs) && visit(a.Pos(), a.Rhs[k])
					}
				}
			}
		}
	}
	return false
}

func hasName(root ast.Node, match func(string) bool) bool {
	return anyIn(root, func(n ast.Node) bool {
		s := name(n)
		return s != "" && match(strings.ToLower(s))
	})
}

func retryName(s string) bool {
	return strings.Contains(s, "retry") || strings.Contains(s, "attempt") || strings.Contains(s, "backoff")
}

func retryLoop(loop *ast.ForStmt) bool {
	return (loop.Init != nil && hasName(loop.Init, retryName)) ||
		(loop.Cond != nil && hasName(loop.Cond, retryName)) ||
		hasName(loop.Body, retryName)
}

func (s source) callback(call *ast.CallExpr, fn *ast.FuncLit) bool {
	if !slices.ContainsFunc(call.Args, func(e ast.Expr) bool { return ast.Unparen(e) == fn }) {
		return false
	}
	p := s.imported(call.Fun)
	switch {
	case strings.HasPrefix(p, "github.com/cenkalti/backoff"):
		return slices.Contains([]string{"Retry", "RetryNotify", "RetryNotifyWithTimer"}, name(call.Fun))
	case p == "github.com/avast/retry-go", strings.HasPrefix(p, "github.com/avast/retry-go/"), p == "github.com/sethvargo/go-retry":
		return name(call.Fun) == "Do" || name(call.Fun) == "DoWithData"
	case p == "github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry", p == "github.com/hashicorp/terraform-plugin-sdk/helper/resource":
		return name(call.Fun) == "Retry" || name(call.Fun) == "RetryContext"
	}
	return false
}
