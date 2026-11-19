package retrycheck

import (
	"go/ast"
	"slices"
	"strings"
)

func (s source) checkWebhook(fn string, typ *ast.FuncType, body *ast.BlockStmt, stack []ast.Node) {
	if !s.httpHandler(typ) {
		return
	}
	hook := s.webhookFile || strings.Contains(strings.ToLower(fn), "webhook") ||
		hasName(body, func(n string) bool {
			return n == "parsewebhook" || n == "validatepayload" || strings.HasPrefix(n, "constructevent")
		})
	for i := len(stack) - 1; !hook && i >= 0; i-- {
		if call, ok := stack[i].(*ast.CallExpr); ok {
			for _, a := range call.Args {
				if strings.Contains(strings.ToLower(s.stringValue(a)), "webhook") {
					hook = true
					break
				}
			}
		}
	}
	if !hook || anyIn(body, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		return ok && slices.Contains([]string{
			"dedup", "dedupe", "deduplicate", "alreadyprocessed", "isduplicate", "isduplicatedelivery",
			"claimevent", "claimdelivery", "inserteventifabsent", "insertifabsent", "setnx", "loadorstore", "onconflict",
		}, strings.ToLower(name(c.Fun)))
	}) {
		return
	}
	var effect *ast.CallExpr
	anyIn(body, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if ok && webhookEffect(name(c.Fun)) {
			effect = c
			return true
		}
		return false
	})
	if effect != nil {
		s.report(webhookRule, effect, "webhook handler performs an effect with no visible delivery dedupe")
	}
}

func (s source) httpHandler(t *ast.FuncType) bool {
	var writer, request bool
	for _, p := range t.Params.List {
		switch e := p.Type.(type) {
		case *ast.SelectorExpr:
			writer = writer || s.imported(e) == "net/http" && name(e) == "ResponseWriter"
			if strings.HasPrefix(s.imported(e), "github.com/labstack/echo") && name(e) == "Context" {
				return true
			}
		case *ast.StarExpr:
			request = request || s.imported(e.X) == "net/http" && name(e.X) == "Request"
			if (s.imported(e.X) == "github.com/gin-gonic/gin" && name(e.X) == "Context") ||
				(strings.HasPrefix(s.imported(e.X), "github.com/gofiber/fiber") && name(e.X) == "Ctx") {
				return true
			}
		}
	}
	return writer && request
}

func webhookEffect(n string) bool {
	if slices.Contains([]string{"Publish", "Dispatch", "Enqueue", "Insert", "Create", "Save", "Send", "Exec", "Execute", "Run", "Charge", "Refund", "Update", "Delete", "Append", "handleHook", "handleEvent", "handleWebhook"}, n) {
		return true
	}
	for _, prefix := range []string{"Publish", "Dispatch", "Enqueue", "Insert", "Send", "Execute", "Run", "Charge", "Refund"} {
		if suffix, ok := strings.CutPrefix(n, prefix); ok && suffix != "" && suffix[0] >= 'A' && suffix[0] <= 'Z' {
			return true
		}
	}
	return false
}
