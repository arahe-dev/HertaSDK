// Package lang implements the Herta contract language: lexer, parser,
// and the contract checker that lifts hertasdk's construction-time
// validation to compile time. See docs/SPEC.md.
package lang

import "fmt"

type TokKind int

const (
	TokEOF TokKind = iota
	TokIdent
	TokInt
	TokDuration // integer + unit suffix (ns, us, ms, s, m, h)
	TokString
	TokLBrace
	TokRBrace
	TokLParen
	TokRParen
	TokComma
	TokColon
	TokColonColon
	TokArrow
	TokSym // other punctuation/operators, only inside handler bodies
)

// Tok is a lexer token with 1-based source position.
type Tok struct {
	Kind TokKind
	Text string
	Line int
	Col  int
}

func (t Tok) String() string {
	return fmt.Sprintf("%s(%q)@%d:%d", kindName(t.Kind), t.Text, t.Line, t.Col)
}

func kindName(k TokKind) string {
	switch k {
	case TokEOF:
		return "EOF"
	case TokIdent:
		return "ident"
	case TokInt:
		return "int"
	case TokDuration:
		return "duration"
	case TokString:
		return "string"
	case TokLBrace:
		return "{"
	case TokRBrace:
		return "}"
	case TokLParen:
		return "("
	case TokRParen:
		return ")"
	case TokComma:
		return ","
	case TokColon:
		return ":"
	case TokColonColon:
		return "::"
	case TokArrow:
		return "->"
	case TokSym:
		return "sym"
	}
	return "?"
}
