package lang

import "strconv"

func (p *parser) word(w string) error {
	t, err := p.ident(w)
	if err != nil {
		return err
	}
	if t.Text != w {
		return p.errf("expected %s, got %q", w, t.Text)
	}
	return nil
}

func (p *parser) compositionClause(op *Op) error {
	t := p.adv()
	switch t.Text {
	case "calls":
		for {
			n, err := p.ident("callee name")
			if err != nil {
				return err
			}
			op.Calls = append(op.Calls, n.Text)
			if p.cur().Kind != TokComma {
				break
			}
			p.adv()
		}
	case "lock":
		n, err := p.ident("lock name")
		if err != nil {
			return err
		}
		l := LockDecl{Name: n.Text}
		if p.cur().Text == "shared" {
			p.adv()
			l.Shared = true
		}
		op.Locks = append(op.Locks, l)
	case "dedupe":
		if err := p.word("key"); err != nil {
			return err
		}
		n, err := p.ident("dedupe parameter")
		if err != nil {
			return err
		}
		op.Dedupe = n.Text
	case "evidence":
		if err := p.word("dispatch"); err != nil {
			return err
		}
		op.TrackDispatch = true
	case "detached":
		op.Detached = true
		if p.cur().Text == "for" {
			p.adv()
			n, err := p.ident("cleanup|shutdown|background")
			if err != nil {
				return err
			}
			switch n.Text {
			case "cleanup", "shutdown", "background":
				op.Owner = n.Text
			default:
				return p.errf("invalid detached owner %q", n.Text)
			}
		}
	case "budget":
		n, err := p.expect(TokInt, "positive attempt budget")
		if err != nil {
			return err
		}
		op.Budget, err = strconv.ParseInt(n.Text, 10, 64)
		if err != nil || op.Budget < 1 {
			return p.errf("budget must be a positive int64")
		}
	case "classify":
		if _, err := p.expect(TokLBrace, "{"); err != nil {
			return err
		}
		for p.cur().Kind != TokRBrace {
			n, err := p.ident("error name or _")
			if err != nil {
				return err
			}
			for _, s := range []string{"=", ">"} {
				if p.cur().Text != s {
					return p.errf("expected =>")
				}
				p.adv()
			}
			o, err := p.ident("classification outcome")
			if err != nil {
				return err
			}
			switch o.Text {
			case "uncertain", "transient", "permanent", "throttled", "success":
			default:
				return p.errf("invalid outcome %q", o.Text)
			}
			op.Classify = append(op.Classify, ClassifyArm{Error: n.Text, Outcome: o.Text})
			if p.cur().Kind != TokComma {
				break
			}
			p.adv()
		}
		if _, err := p.expect(TokRBrace, "}"); err != nil {
			return err
		}
		if op.Classify == nil {
			op.Classify = []ClassifyArm{}
		}
	}
	return nil
}
