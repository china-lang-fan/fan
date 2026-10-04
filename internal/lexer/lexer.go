package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"fan/internal/token"
)

type Lexer struct {
	src     []rune
	pos     int
	line    int
	col     int
	prevTok token.Type
}

func New(source string) *Lexer {
	return &Lexer{
		src:     []rune(source),
		pos:     0,
		line:    1,
		col:     1,
		prevTok: token.NEWLINE,
	}
}

func (l *Lexer) NextToken() token.Token {
	l.skipSpaces()
	if l.pos >= len(l.src) {
		return l.emit(token.EOF, "")
	}

	r := l.src[l.pos]

	if r == '\n' {
		tok := l.emit(token.NEWLINE, "\n")
		l.advance()
		l.line++
		l.col = 1
		return tok
	}

	if r == '#' {
		l.consumeLineComment()
		return l.NextToken()
	}

	if r == '"' {
		return l.readString()
	}

	if isDigit(r) {
		return l.readNumber()
	}

	if isIdentStart(r) {
		return l.readIdentifierOrKeyword()
	}

	switch r {
	case '(', '（':
		return l.emitAndAdvance(token.LPAREN, string(r))
	case ')', '）':
		return l.emitAndAdvance(token.RPAREN, string(r))
	case '[':
		return l.emitAndAdvance(token.LBRACK, "[")
	case ']':
		return l.emitAndAdvance(token.RBRACK, "]")
	case '{':
		return l.emitAndAdvance(token.LBRACE, "{")
	case '}':
		return l.emitAndAdvance(token.RBRACE, "}")
	case ',':
		return l.emitAndAdvance(token.COMMA, ",")
	case '，':
		return l.emitAndAdvance(token.COMMA, "，")
	case '@', '＠':
		return l.emitAndAdvance(token.TAG, string(r))
	case '？', '?':
		return l.emitAndAdvance(token.QUEST, string(r))
	case '.':
		return l.emitAndAdvance(token.DOT, ".")
	case ':':
		return l.emitAndAdvance(token.COLON, ":")
	case '：':
		return l.emitAndAdvance(token.COLON, "：")
	case '、':
		return l.emitAndAdvance(token.COMMA, "、")
	case '+':
		if l.peek(1) == '=' {
			return l.emitAndAdvanceN(token.PLUS_EQ, "+=", 2)
		}
		if l.peek(1) == '+' {
			return l.emitAndAdvanceN(token.INC, "++", 2)
		}
		return l.emitAndAdvance(token.PLUS, "+")
	case '-':
		if l.peek(1) == '>' {
			return l.emitAndAdvanceN(token.ARROW, "->", 2)
		}
		if l.peek(1) == '=' {
			return l.emitAndAdvanceN(token.MINUS_EQ, "-=", 2)
		}
		if l.peek(1) == '-' {
			return l.emitAndAdvanceN(token.DEC, "--", 2)
		}
		return l.emitAndAdvance(token.MINUS, "-")
	case '→':
		return l.emitAndAdvance(token.ARROW, "→")
	case '*':
		if l.peek(1) == '=' {
			return l.emitAndAdvanceN(token.STAR_EQ, "*=", 2)
		}
		return l.emitAndAdvance(token.STAR, "*")
	case '/':
		if l.peek(1) == '=' {
			return l.emitAndAdvanceN(token.SLASH_EQ, "/=", 2)
		}
		return l.emitAndAdvance(token.SLASH, "/")
	case '%':
		return l.emitAndAdvance(token.PERCENT, "%")
	case '=':
		if l.peek(1) == '=' {
			return l.emitAndAdvanceN(token.EQ, "==", 2)
		}
		return l.emitAndAdvance(token.ASSIGN, "=")
	case '!':
		if l.peek(1) == '=' {
			return l.emitAndAdvanceN(token.NEQ, "!=", 2)
		}
		return l.emitAndAdvance(token.NOT, "!")
	case '<':
		if l.peek(1) == '=' {
			return l.emitAndAdvanceN(token.LTE, "<=", 2)
		}
		return l.emitAndAdvance(token.LT, "<")
	case '>':
		if l.peek(1) == '=' {
			return l.emitAndAdvanceN(token.GTE, ">=", 2)
		}
		return l.emitAndAdvance(token.GT, ">")
	case '&':
		if l.peek(1) == '&' {
			return l.emitAndAdvanceN(token.AND, "&&", 2)
		}
	case '|':
		if l.peek(1) == '|' {
			return l.emitAndAdvanceN(token.OR, "||", 2)
		}
	}

	tok := l.emit(token.ILLEGAL, string(r))
	l.advance()
	return tok
}

func (l *Lexer) skipSpaces() {
	for l.pos < len(l.src) {
		r := l.src[l.pos]
		if r == ' ' || r == '\t' || r == '\r' {
			l.advance()
			continue
		}
		break
	}
}

func (l *Lexer) consumeLineComment() {
	for l.pos < len(l.src) && l.src[l.pos] != '\n' {
		l.advance()
	}
}

func (l *Lexer) advance() {
	if l.pos < len(l.src) {
		l.pos++
		l.col++
	}
}

func (l *Lexer) peek(offset int) rune {
	i := l.pos + offset
	if i < 0 || i >= len(l.src) {
		return 0
	}
	return l.src[i]
}

func (l *Lexer) emit(t token.Type, literal string) token.Token {
	tok := token.Token{Type: t, Literal: literal, Line: l.line, Column: l.col}
	if t != token.EOF {
		l.prevTok = t
	}
	return tok
}

func (l *Lexer) emitAndAdvance(t token.Type, literal string) token.Token {
	tok := l.emit(t, literal)
	l.advance()
	return tok
}

func (l *Lexer) emitAndAdvanceN(t token.Type, literal string, n int) token.Token {
	tok := l.emit(t, literal)
	for i := 0; i < n; i++ {
		l.advance()
	}
	return tok
}

func (l *Lexer) readNumber() token.Token {
	start := l.pos
	startCol := l.col
	for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
		l.advance()
	}
	isFloat := false
	if l.pos < len(l.src) && l.src[l.pos] == '.' && l.pos+1 < len(l.src) && isDigit(l.src[l.pos+1]) {
		isFloat = true
		l.advance()
		for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
			l.advance()
		}
	}
	literal := string(l.src[start:l.pos])
	t := token.INT
	if isFloat {
		t = token.FLOAT
	}
	return token.Token{Type: t, Literal: literal, Line: l.line, Column: startCol}
}

func (l *Lexer) readString() token.Token {
	startCol := l.col
	l.advance()
	var b strings.Builder
	for l.pos < len(l.src) {
		r := l.src[l.pos]
		if r == '"' {
			l.advance()
			return token.Token{Type: token.STRING, Literal: b.String(), Line: l.line, Column: startCol}
		}
		if r == '\n' {
			return token.Token{Type: token.ILLEGAL, Literal: "字符串未闭合", Line: l.line, Column: startCol}
		}
		if r == '\\' {
			next := l.peek(1)
			switch next {
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
				return token.Token{Type: token.ILLEGAL, Literal: fmt.Sprintf("未知转义 \\%s", string(next)), Line: l.line, Column: l.col}
			}
			l.advance()
			l.advance()
			continue
		}
		b.WriteRune(r)
		l.advance()
	}
	return token.Token{Type: token.ILLEGAL, Literal: "字符串未闭合", Line: l.line, Column: startCol}
}

func (l *Lexer) readIdentifierOrKeyword() token.Token {
	start := l.pos
	startCol := l.col
	for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
		l.advance()
	}
	literal := string(l.src[start:l.pos])

	if literal == "注释" {
		next := rune(0)
		if l.pos < len(l.src) {
			next = l.src[l.pos]
		}
		if next == 0 || next == '\n' || next == ' ' || next == '\t' || next == ':' || next == '：' {
			l.consumeLineComment()
			return l.NextToken()
		}
	}

	if t, ok := token.Lookup(literal); ok {
		return token.Token{Type: t, Literal: literal, Line: l.line, Column: startCol}
	}
	return token.Token{Type: token.IDENT, Literal: literal, Line: l.line, Column: startCol}
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isIdentStart(r rune) bool {
	if r == '_' {
		return true
	}
	if unicode.IsLetter(r) {
		return true
	}
	return false
}

func isIdentPart(r rune) bool {
	if isIdentStart(r) {
		return true
	}
	return isDigit(r)
}

var _ = utf8.RuneError
