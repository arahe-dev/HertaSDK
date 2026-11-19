package lang

import (
	"fmt"
	"strings"
	"time"
)

type callEdge struct {
	name     string
	args     []Tok
	explicit bool
}

func opName(op *Op) string { return op.Runtime + "::" + op.Name }

// Extract from tokens, not source regexes: quoted strings cannot invent edges.
// RawStmt text is spaced by the parser and may span multiple source lines.
func handlerEdges(h *Handler) []callEdge {
	var body strings.Builder
	for _, s := range h.Body {
		switch s := s.(type) {
		case *RawStmt:
			body.WriteString(s.Text)
			body.WriteByte('\n')
		case *FailStmt:
			body.WriteString(s.Arg)
			body.WriteByte('\n')
		}
	}
	ts, err := Lex(body.String())
	if err != nil {
		return nil
	}
	params := map[string]string{}
	for _, p := range h.Params {
		if strings.HasSuffix(p.Type, "Op") {
			params[p.Name] = strings.TrimSuffix(p.Type, "Op")
		}
	}
	var edges []callEdge
	for i := 0; i+2 < len(ts); i++ {
		if ts[i].Kind != TokIdent {
			continue
		}
		name, open, explicit := "", 0, false
		if ts[i].Text == "call" && ts[i+1].Kind == TokIdent && ts[i+2].Kind == TokLParen {
			name, open, explicit = ts[i+1].Text, i+2, true
		} else if i+3 < len(ts) && params[ts[i].Text] != "" && ts[i+1].Text == "." && ts[i+2].Text == "Do" && ts[i+3].Kind == TokLParen {
			name, open = params[ts[i].Text], i+3
		}
		if name == "" {
			continue
		}
		depth, end := 1, open+1
		for ; end < len(ts) && depth > 0; end++ {
			if ts[end].Kind == TokLParen {
				depth++
			}
			if ts[end].Kind == TokRParen {
				depth--
			}
		}
		if depth == 0 {
			edges = append(edges, callEdge{name: name, args: ts[open+1 : end-1], explicit: explicit})
		}
	}
	return edges
}

// Key namespaces default to runtime + parameter name. Simple forwarding of a
// key through call arguments also preserves the caller's namespace.
func edgeKeys(edge callEdge, callee *Op, keys map[string]string) map[string]string {
	out := map[string]string{}
	for _, p := range callee.Params {
		out[p.Name] = callee.Runtime + "/" + p.Name
	}
	if edge.explicit {
		start, depth, index := 0, 0, 0
		for i := 0; i <= len(edge.args); i++ {
			if i < len(edge.args) {
				switch edge.args[i].Kind {
				case TokLParen, TokLBrace:
					depth++
				case TokRParen, TokRBrace:
					depth--
				}
			}
			if i == len(edge.args) || (edge.args[i].Kind == TokComma && depth == 0) {
				if index < len(callee.Params) && i-start == 1 && edge.args[start].Kind == TokIdent {
					if k := keys[edge.args[start].Text]; k != "" {
						out[callee.Params[index].Name] = k
					}
				}
				start, index = i+1, index+1
			}
		}
	} else {
		for i := 0; i+2 < len(edge.args); i++ {
			if edge.args[i].Kind != TokIdent || edge.args[i+1].Kind != TokColon || edge.args[i+2].Kind != TokIdent {
				continue
			}
			for _, p := range callee.Params {
				field := strings.ReplaceAll(p.Name, "_", "")
				if strings.EqualFold(field, edge.args[i].Text) && keys[edge.args[i+2].Text] != "" {
					out[p.Name] = keys[edge.args[i+2].Text]
				}
			}
		}
	}
	return out
}

func checkComposition(prog *Program, ops map[string]*Op, rts map[string]*Runtime, at func(Diagnostic)) {
	edges := map[string][]callEdge{}
	ambiguous := map[string]bool{}
	var ordered []*Op
	for _, d := range prog.Decls {
		switch d := d.(type) {
		case *ErrorDecl:
			ambiguous[d.Name] = true
		case *Op:
			if ops[opName(d)] != d {
				continue
			}
			ordered = append(ordered, d)
			for _, n := range d.Calls {
				edges[opName(d)] = append(edges[opName(d)], callEdge{name: n})
			}
		case *Handler:
			edges[d.Runtime+"::"+d.Name] = append(edges[d.Runtime+"::"+d.Name], handlerEdges(d)...)
		}
	}
	effectRank := map[string]int{"pure": 0, "idempotent": 1, "nonidempotent": 2}
	effective := map[*Op]int{}
	for _, op := range ordered {
		effective[op] = effectRank[op.Effect]
	}
	for changed := true; changed; {
		changed = false
		for _, op := range ordered {
			for _, edge := range edges[opName(op)] {
				if child := effective[ops[op.Runtime+"::"+edge.name]]; child > effective[op] {
					effective[op] = child
					changed = true
				}
			}
		}
	}
	seen := map[string]bool{}
	report := func(op *Op, code, path, msg string) {
		text := code + " path " + path + ": " + msg
		if seen[text] {
			return
		}
		seen[text] = true
		at(Diagnostic{op.Line, op.Col, text})
	}
	for _, root := range ordered {
		rootName := opName(root)
		if root.Dedupe != "" {
			found := false
			for _, p := range root.Params {
				if p.Name == root.Dedupe {
					found = true
				}
			}
			if !found {
				report(root, "HC014", rootName, "dedupe key is not a parameter; declare a dedupe key parameter")
			}
		}
		if root.Classify != nil {
			fallback, transient := false, false
			for _, arm := range root.Classify {
				if arm.Error == "_" {
					fallback = true
				}
				if arm.Outcome == "uncertain" && transient {
					report(root, "HC010", rootName, "transient arm precedes uncertain arm; put uncertain arms first for joined errors")
				}
				if arm.Outcome == "transient" {
					transient = true
				}
			}
			if !fallback {
				report(root, "HC009", rootName, "classification has no _ arm; add _ => permanent")
			}
		}
		budget := root.Budget
		if budget == 0 {
			budget = 10
		}
		visits := 0
		var walk func(*Op, []*Op, []map[string]string, int64)
		walk = func(op *Op, ancestors []*Op, keyStack []map[string]string, attempts int64) {
			visits++
			if visits > 10000 {
				report(root, "HC015", rootName, "composition traversal exceeds 10000 paths; split the graph or reduce declared calls")
				return
			}
			pathOps := append(append([]*Op{}, ancestors...), op)
			names := make([]string, len(pathOps))
			for i, p := range pathOps {
				names[i] = opName(p)
			}
			path := strings.Join(names, " -> ")
			keys := keyStack[len(keyStack)-1]
			for i, held := range ancestors {
				if held.Key != "" && op.Key != "" && (held == op || (held.Runtime == op.Runtime && keyStack[i][held.Key] == keys[op.Key])) {
					report(root, "HC001", path, "serialize key reacquired while held by "+opName(held)+"; release the caller key or use a distinct key namespace")
				}
				for _, a := range held.Locks {
					for _, b := range op.Locks {
						if a.Name != b.Name {
							continue
						}
						kind := "exclusive self-deadlock"
						if a.Shared && b.Shared {
							kind = "recursive RLock writer-starvation deadlock"
						}
						report(root, "HC002", path, "lock "+a.Name+" reacquired: "+kind+"; release the caller lock or remove nested acquisition")
					}
				}
			}
			if len(ancestors) > 0 {
				used := map[string]uint64{}
				for _, p := range pathOps {
					if p.Runtime == op.Runtime {
						for _, r := range p.Resources {
							if r.Units > 0 {
								units := uint64(r.Units)
								if ^uint64(0)-used[r.Name] < units {
									used[r.Name] = ^uint64(0)
								} else {
									used[r.Name] += units
								}
							}
						}
					}
				}
				if rt := rts[op.Runtime]; rt != nil {
					for _, r := range rt.Resources {
						if r.Capacity > 0 && used[r.Name] > uint64(r.Capacity) {
							report(root, "HC003", path, fmt.Sprintf("held resource %s units %d exceed capacity %d; release caller units or increase capacity", r.Name, used[r.Name], r.Capacity))
						}
					}
				}
			}
			if op.Effect == "nonidempotent" {
				if root.Effect == "pure" || root.Effect == "idempotent" {
					report(root, "HC004", path, "declared "+root.Effect+" has effective effect nonidempotent; declare nonidempotent or remove the effectful call")
				}
				unsafeRetry := false
				if root.Retry != nil {
					for _, o := range root.Retry.On {
						if o.Name == "uncertain" {
							unsafeRetry = true
						}
					}
					for errName := range ambiguous {
						class := classified(root, errName)
						for _, o := range root.Retry.On {
							if o.Name == class {
								unsafeRetry = true
							}
						}
					}
				}
				// Preserve the existing local unsafe-retry diagnostic and golden.
				localLegacy := root == op && root.Effect == "nonidempotent" && retries(root, "uncertain")
				if unsafeRetry && root.Dedupe == "" && !localLegacy {
					report(root, "HC005", path, "retry can replay an effective nonidempotent effect; add dedupe key <param> or stop retrying uncertain/ambiguous outcomes")
				}
				for _, arm := range root.Classify {
					if arm.Outcome == "transient" && (arm.Error == "_" || ambiguous[arm.Error]) {
						report(root, "HC011", path, "ambiguous error (or _) classified transient with nonidempotent effect; classify it uncertain or permanent")
					}
				}
			}
			max := int64(1)
			if op.Retry != nil && op.Retry.Max > 1 {
				max = op.Retry.Max
			}
			if attempts > budget/max {
				report(root, "HC006", path, fmt.Sprintf("retry attempt product exceeds budget %d; reduce retry max or set the outermost budget", budget))
				attempts = budget
			} else {
				attempts *= max
			}
			if len(ancestors) > 0 {
				caller := ancestors[len(ancestors)-1]
				if !op.Detached {
					childTimeout, e := time.ParseDuration(op.Timeout)
					for i := len(ancestors) - 1; i >= 0; i-- {
						parent := ancestors[i]
						parentTimeout, pe := time.ParseDuration(parent.Timeout)
						if e == nil && pe == nil && childTimeout > parentTimeout {
							report(root, "HC007", path, "callee timeout exceeds inherited timeout from "+opName(parent)+"; shorten callee timeout or increase caller timeout")
						}
						if parent.Detached {
							break
						}
					}
				}
				if op.Detached && op.Owner == "" && effective[op] > 0 {
					report(root, "HC008", path, "non-pure detached work drops cancellation from "+opName(caller)+"; declare detached for <owner> or propagate cancellation")
				}
			}
			for _, a := range ancestors {
				if a == op {
					report(root, "HC012", path, "call cycle; remove recursion or break the composition edge")
					return
				}
			}
			for _, edge := range edges[opName(op)] {
				callee := ops[op.Runtime+"::"+edge.name]
				if callee == nil {
					report(root, "HC013", path+" -> "+edge.name, "unknown callee; declare the op in the same runtime or correct the call")
					continue
				}
				childKeys := edgeKeys(edge, callee, keys)
				walk(callee, pathOps, append(append([]map[string]string{}, keyStack...), childKeys), attempts)
			}
		}
		keys := map[string]string{}
		for _, p := range root.Params {
			keys[p.Name] = root.Runtime + "/" + p.Name
		}
		walk(root, nil, []map[string]string{keys}, 1)
	}
}

func classified(op *Op, name string) string {
	for _, arm := range op.Classify {
		if arm.Error == name || arm.Error == "_" {
			return arm.Outcome
		}
	}
	return ""
}

func retries(op *Op, class string) bool {
	if op.Retry != nil {
		for _, o := range op.Retry.On {
			if o.Name == class {
				return true
			}
		}
	}
	return false
}
