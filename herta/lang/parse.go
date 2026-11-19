package lang

import (
	"fmt"
	"strconv"
	"strings"
)

type parser struct {
	toks []Tok
	i    int
}

// Parse lexes and parses a Herta source file.
func Parse(src string) (*Program, error) {
	toks, err := Lex(src)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	prog := &Program{}
	for p.cur().Kind != TokEOF {
		d, err := p.decl()
		if err != nil {
			return nil, err
		}
		prog.Decls = append(prog.Decls, d)
	}
	return prog, nil
}

func (p *parser) cur() Tok { return p.toks[p.i] }
func (p *parser) adv() Tok { t := p.toks[p.i]; p.i++; return t }

func (p *parser) errf(format string, args ...any) error {
	t := p.cur()
	return fmt.Errorf("%d:%d: %s", t.Line, t.Col, fmt.Sprintf(format, args...))
}

func (p *parser) expect(k TokKind, what string) (Tok, error) {
	t := p.cur()
	if t.Kind != k {
		return t, p.errf("expected %s, got %s", what, t.String())
	}
	p.i++
	return t, nil
}

func (p *parser) ident(what string) (Tok, error) {
	return p.expect(TokIdent, what)
}

func (p *parser) decl() (Decl, error) {
	t := p.cur()
	if t.Kind != TokIdent {
		return nil, p.errf("expected declaration (runtime|op|handler), got %s", t.String())
	}
	switch t.Text {
	case "runtime":
		return p.runtimeDecl()
	case "op":
		return p.opDecl()
	case "handler":
		return p.handlerDecl()
	case "error":
		kw := p.adv()
		n, err := p.ident("error name")
		if err != nil {
			return nil, err
		}
		if err := p.word("ambiguous"); err != nil {
			return nil, err
		}
		return &ErrorDecl{Name: n.Text, Line: kw.Line, Col: kw.Col}, nil
	}
	return nil, p.errf("unknown declaration %q", t.Text)
}

func (p *parser) runtimeDecl() (*Runtime, error) {
	kw := p.adv() // runtime
	name, err := p.ident("runtime name")
	if err != nil {
		return nil, err
	}
	rt := &Runtime{Name: name.Text, Line: kw.Line, Col: kw.Col}
	if _, err := p.expect(TokLBrace, "{"); err != nil {
		return nil, err
	}
	for p.cur().Kind != TokRBrace {
		k, err := p.ident("resource")
		if err != nil {
			return nil, err
		}
		if k.Text != "resource" {
			return nil, p.errf("expected resource, got %q", k.Text)
		}
		rn, err := p.ident("resource name")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokIdent, "capacity"); err != nil {
			return nil, err
		}
		if p.toks[p.i-1].Text != "capacity" {
			return nil, p.errf("expected capacity, got %q", p.toks[p.i-1].Text)
		}
		capTok, err := p.expect(TokInt, "capacity value")
		if err != nil {
			return nil, err
		}
		cap_, _ := strconv.ParseInt(capTok.Text, 10, 64)
		rt.Resources = append(rt.Resources, ResourceDecl{
			Name: rn.Text, Capacity: cap_, Line: rn.Line, Col: rn.Col,
		})
	}
	p.adv() // }
	return rt, nil
}

func (p *parser) qualName(what string) (runtime, name string, err error) {
	rt, err := p.ident("runtime name")
	if err != nil {
		return "", "", err
	}
	if _, err := p.expect(TokColonColon, "::"); err != nil {
		return "", "", err
	}
	nm, err := p.ident(what)
	if err != nil {
		return "", "", err
	}
	return rt.Text, nm.Text, nil
}

func (p *parser) params() ([]Param, error) {
	if _, err := p.expect(TokLParen, "("); err != nil {
		return nil, err
	}
	var ps []Param
	if p.cur().Kind == TokRParen {
		p.adv()
		return ps, nil
	}
	for {
		n, err := p.ident("parameter name")
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokColon, ":"); err != nil {
			return nil, err
		}
		ty, err := p.ident("parameter type")
		if err != nil {
			return nil, err
		}
		ps = append(ps, Param{Name: n.Text, Type: ty.Text})
		if p.cur().Kind == TokComma {
			p.adv()
			continue
		}
		break
	}
	if _, err := p.expect(TokRParen, ")"); err != nil {
		return nil, err
	}
	return ps, nil
}

func (p *parser) opDecl() (*Op, error) {
	kw := p.adv() // op
	rt, nm, err := p.qualName("operation name")
	if err != nil {
		return nil, err
	}
	op := &Op{Runtime: rt, Name: nm, Line: kw.Line, Col: kw.Col}
	if op.Params, err = p.params(); err != nil {
		return nil, err
	}
	if _, err := p.expect(TokArrow, "->"); err != nil {
		return nil, err
	}
	res, err := p.ident("result type")
	if err != nil {
		return nil, err
	}
	op.Result = res.Text
	if _, err := p.expect(TokLBrace, "{"); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for p.cur().Kind != TokRBrace {
		t := p.cur()
		if t.Kind != TokIdent {
			return nil, p.errf("expected contract clause, got %s", t.String())
		}
		if seen[t.Text] && t.Text != "lock" {
			return nil, p.errf("duplicate %q clause", t.Text)
		}
		seen[t.Text] = true
		switch t.Text {
		case "calls", "lock", "dedupe", "detached", "budget", "classify", "evidence":
			if err := p.compositionClause(op); err != nil {
				return nil, err
			}
		case "effect":
			p.adv()
			v, err := p.ident("effect value")
			if err != nil {
				return nil, err
			}
			switch v.Text {
			case "pure", "idempotent", "nonidempotent":
				op.Effect = v.Text
			default:
				return nil, p.errf("invalid effect %q (want pure|idempotent|nonidempotent)", v.Text)
			}
		case "admission":
			p.adv()
			v, err := p.ident("admission value")
			if err != nil {
				return nil, err
			}
			switch v.Text {
			case "wait", "reject":
				op.Admission = v.Text
			default:
				return nil, p.errf("invalid admission %q (want wait|reject)", v.Text)
			}
		case "resources":
			p.adv()
			if _, err := p.expect(TokLBrace, "{"); err != nil {
				return nil, err
			}
			for {
				rn, err := p.ident("resource name")
				if err != nil {
					return nil, err
				}
				u, err := p.ident("units")
				if err != nil {
					return nil, err
				}
				if u.Text != "units" {
					return nil, p.errf("expected units, got %q", u.Text)
				}
				uv, err := p.expect(TokInt, "units value")
				if err != nil {
					return nil, err
				}
				n, _ := strconv.ParseInt(uv.Text, 10, 64)
				op.Resources = append(op.Resources, OpResource{
					Name: rn.Text, Units: n, Line: rn.Line, Col: rn.Col,
				})
				if p.cur().Kind == TokComma {
					p.adv()
					continue
				}
				break
			}
			if _, err := p.expect(TokRBrace, "}"); err != nil {
				return nil, err
			}
		case "timeout":
			p.adv()
			v, err := p.expect(TokDuration, "duration (e.g. 500ms)")
			if err != nil {
				return nil, err
			}
			op.Timeout = v.Text
		case "serialize":
			p.adv()
			k, err := p.ident("key")
			if err != nil {
				return nil, err
			}
			if k.Text != "key" {
				return nil, p.errf("expected key, got %q", k.Text)
			}
			kn, err := p.ident("key parameter")
			if err != nil {
				return nil, err
			}
			op.Key = kn.Text
		case "retry":
			p.adv()
			m, err := p.ident("max")
			if err != nil {
				return nil, err
			}
			if m.Text != "max" {
				return nil, p.errf("expected max, got %q", m.Text)
			}
			mv, err := p.expect(TokInt, "max attempts")
			if err != nil {
				return nil, err
			}
			rd := &RetryDecl{Line: t.Line, Col: t.Col}
			rd.Max, _ = strconv.ParseInt(mv.Text, 10, 64)
			if _, err := p.ident("on"); err != nil {
				return nil, err
			}
			if p.toks[p.i-1].Text != "on" {
				return nil, p.errf("expected on, got %q", p.toks[p.i-1].Text)
			}
			if _, err := p.expect(TokLBrace, "{"); err != nil {
				return nil, err
			}
			for {
				o, err := p.ident("outcome")
				if err != nil {
					return nil, err
				}
				switch o.Text {
				case "transient", "uncertain", "permanent", "throttled", "success":
					rd.On = append(rd.On, OutcomeRef{Name: o.Text, Line: o.Line, Col: o.Col})
				default:
					return nil, p.errf("invalid outcome %q", o.Text)
				}
				if p.cur().Kind == TokComma {
					p.adv()
					continue
				}
				break
			}
			if _, err := p.expect(TokRBrace, "}"); err != nil {
				return nil, err
			}
			op.Retry = rd
		default:
			return nil, p.errf("unknown clause %q", t.Text)
		}
	}
	p.adv() // }
	return op, nil
}

func (p *parser) handlerDecl() (*Handler, error) {
	kw := p.adv() // handler
	rt, nm, err := p.qualName("operation name")
	if err != nil {
		return nil, err
	}
	h := &Handler{Runtime: rt, Name: nm, Line: kw.Line, Col: kw.Col}
	if h.Params, err = p.params(); err != nil {
		return nil, err
	}
	if _, err := p.expect(TokArrow, "->"); err != nil {
		return nil, err
	}
	if _, err := p.expect(TokLParen, "("); err != nil {
		return nil, err
	}
	for {
		ty, err := p.ident("result type")
		if err != nil {
			return nil, err
		}
		h.Results = append(h.Results, ty.Text)
		if p.cur().Kind == TokComma {
			p.adv()
			continue
		}
		break
	}
	if _, err := p.expect(TokRParen, ")"); err != nil {
		return nil, err
	}
	if _, err := p.expect(TokLBrace, "{"); err != nil {
		return nil, err
	}
	// Body: one statement per source line. A line starting with `fail`
	// is the only contract-relevant statement; the rest are opaque text.
	// Braces open nested blocks whose lines are statements too.
	depth := 0
	for {
		t := p.cur()
		if t.Kind == TokEOF || (t.Kind == TokRBrace && depth == 0) {
			break
		}
		line, col := t.Line, t.Col
		if t.Kind == TokIdent && t.Text == "fail" {
			p.adv()
			o, err := p.ident("fail outcome")
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokComma, ", after fail outcome"); err != nil {
				return nil, err
			}
			var b strings.Builder
			startLine := p.cur().Line
			for p.cur().Kind != TokEOF && p.cur().Line == startLine {
				tok := p.cur()
				if tok.Kind == TokRBrace && depth == 0 {
					break
				}
				if tok.Kind == TokLBrace {
					depth++
				}
				if tok.Kind == TokRBrace {
					depth--
				}
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(tokText(p.adv()))
			}
			h.Body = append(h.Body, &FailStmt{Outcome: o.Text, Arg: b.String(), Line: line, Col: col})
			continue
		}
		var b strings.Builder
		var fails []OutcomeRef
		startLine := t.Line
		for p.cur().Kind != TokEOF && p.cur().Line == startLine {
			tok := p.cur()
			if tok.Kind == TokRBrace && depth == 0 {
				break
			}
			if tok.Kind == TokLBrace {
				depth++
			}
			if tok.Kind == TokRBrace {
				depth--
			}
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(tokText(p.adv()))
			// A `fail <outcome>` buried in an inline block still returns a
			// classified error; record it so the checker sees it. Strings
			// and comments can't reach here as idents. The outcome may be
			// any token — the checker reports invalid ones.
			if tok.Kind == TokIdent && tok.Text == "fail" &&
				p.cur().Line == startLine && p.cur().Kind != TokRBrace && p.cur().Kind != TokEOF {
				o := p.adv()
				b.WriteByte(' ')
				b.WriteString(tokText(o))
				fails = append(fails, OutcomeRef{Name: o.Text, Line: tok.Line, Col: tok.Col})
			}
		}
		h.Body = append(h.Body, &RawStmt{Text: b.String(), Fails: fails, Line: line, Col: col})
	}
	if _, err := p.expect(TokRBrace, "} for handler body"); err != nil {
		return nil, err
	}
	return h, nil
}

// tokText returns the token's source text; strings keep their quotes so
// preserved handler statements survive a round trip.
func tokText(t Tok) string {
	if t.Kind == TokString {
		return strconv.Quote(t.Text)
	}
	return t.Text
}
