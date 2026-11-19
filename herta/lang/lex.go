package lang

import (
	"fmt"
	"strings"
	"unicode"
)

type lexer struct {
	src  string
	pos  int
	line int
	col  int
	toks []Tok
	err  error
}

// Lex tokenizes a Herta source file. It returns all tokens (ending with
// TokEOF) or the first lexical error.
func Lex(src string) ([]Tok, error) {
	l := &lexer{src: src, line: 1, col: 1}
	for {
		l.skipSpace()
		if l.pos >= len(l.src) {
			l.push(TokEOF, "")
			return l.toks, l.err
		}
		if l.err != nil {
			return nil, l.err
		}
		l.step()
	}
}

func (l *lexer) push(k TokKind, text string) {
	l.toks = append(l.toks, Tok{Kind: k, Text: text, Line: l.line, Col: l.col})
}

func (l *lexer) pushAt(k TokKind, text string, line, col int) {
	l.toks = append(l.toks, Tok{Kind: k, Text: text, Line: line, Col: col})
}

func (l *lexer) failf(format string, args ...any) {
	l.err = fmt.Errorf("%d:%d: %s", l.line, l.col, fmt.Sprintf(format, args...))
}

func (l *lexer) peek() byte {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *lexer) next() byte {
	c := l.src[l.pos]
	l.pos++
	if c == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return c
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '/' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '/' {
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.pos++
			}
			continue
		}
		if unicode.IsSpace(rune(c)) {
			l.next()
			continue
		}
		return
	}
}

func isIdentStart(c byte) bool { return c == '_' || unicode.IsLetter(rune(c)) }
func isIdentChar(c byte) bool {
	return c == '_' || unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c))
}

func (l *lexer) step() {
	line, col := l.line, l.col
	c := l.peek()
	switch {
	case isIdentStart(c):
		start := l.pos
		for l.pos < len(l.src) && isIdentChar(l.peek()) {
			l.next()
		}
		l.pushAt(TokIdent, l.src[start:l.pos], line, col)
	case unicode.IsDigit(rune(c)):
		start := l.pos
		for l.pos < len(l.src) && unicode.IsDigit(rune(l.peek())) {
			l.next()
		}
		// optional duration unit
		unitStart := l.pos
		for l.pos < len(l.src) && isIdentChar(l.peek()) {
			l.next()
		}
		text := l.src[start:l.pos]
		if unit := l.src[unitStart:l.pos]; unit != "" {
			switch unit {
			case "ns", "us", "ms", "s", "m", "h":
				l.pushAt(TokDuration, text, line, col)
			default:
				l.failf("unknown unit suffix %q", unit)
			}
			return
		}
		l.pushAt(TokInt, text, line, col)
	case c == '"':
		l.next()
		var b strings.Builder
		for {
			if l.pos >= len(l.src) {
				l.failf("unterminated string")
				return
			}
			ch := l.next()
			if ch == '"' {
				l.pushAt(TokString, b.String(), line, col)
				return
			}
			if ch == '\\' {
				if l.pos >= len(l.src) {
					l.failf("unterminated string")
					return
				}
				e := l.next()
				switch e {
				case 'n':
					b.WriteByte('\n')
				case 't':
					b.WriteByte('\t')
				case 'r':
					b.WriteByte('\r')
				case '"':
					b.WriteByte('"')
				case '\\':
					b.WriteByte('\\')
				default:
					l.failf("bad escape \\%c", e)
					return
				}
			} else {
				b.WriteByte(ch)
			}
		}
	case c == '{':
		l.next()
		l.pushAt(TokLBrace, "{", line, col)
	case c == '}':
		l.next()
		l.pushAt(TokRBrace, "}", line, col)
	case c == '(':
		l.next()
		l.pushAt(TokLParen, "(", line, col)
	case c == ')':
		l.next()
		l.pushAt(TokRParen, ")", line, col)
	case c == ',':
		l.next()
		l.pushAt(TokComma, ",", line, col)
	case c == ':':
		l.next()
		if l.peek() == '=' {
			l.next()
			l.pushAt(TokSym, ":=", line, col)
		} else if l.peek() == ':' {
			l.next()
			l.pushAt(TokColonColon, "::", line, col)
		} else {
			l.pushAt(TokColon, ":", line, col)
		}
	case c == '-':
		l.next()
		if l.peek() == '>' {
			l.next()
			l.pushAt(TokArrow, "->", line, col)
		} else {
			l.pushAt(TokSym, "-", line, col)
		}
	default:
		// Operators and other punctuation (handler bodies only).
		start := l.pos
		l.next()
		if l.pos < len(l.src) {
			two := l.src[start : l.pos+1]
			switch two {
			case ":=", "==", "!=", "<=", ">=", "&&", "||", "++", "--", "+=", "-=", "*=", "/=":
				l.next()
				l.pushAt(TokSym, two, line, col)
				return
			}
		}
		l.pushAt(TokSym, l.src[start:l.pos], line, col)
	}
}
