package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"fan/internal/token"
)

type Lexer struct {
	src         []rune
	pos         int
	line        int
	col         int
	prevTok     token.Type
	spaceBefore bool
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
	l.spaceBefore = l.skipSpaces()
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

	if r == '\'' || r == '‘' {
		return l.readChar()
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

func (l *Lexer) skipSpaces() bool {
	saw := false
	for l.pos < len(l.src) {
		r := l.src[l.pos]
		if r == ' ' || r == '\t' || r == '\r' {
			l.advance()
			saw = true
			continue
		}
		break
	}
	return saw
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
	tok := token.Token{Type: t, Literal: literal, Line: l.line, Column: l.col, LeadingSpace: l.spaceBefore}
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
			escaped, ok := escapeRune(next)
			if !ok {
				return token.Token{Type: token.ILLEGAL, Literal: fmt.Sprintf("未知转义 \\%s", string(next)), Line: l.line, Column: l.col}
			}
			b.WriteRune(escaped)
			l.advance()
			l.advance()
			continue
		}
		b.WriteRune(r)
		l.advance()
	}
	return token.Token{Type: token.ILLEGAL, Literal: "字符串未闭合", Line: l.line, Column: startCol}
}

// readChar 读取字符字面量，支持半角单引号（'A'）与全角单引号（‘中’）。
// 字面量内容是单个 Unicode 字符（转义后）。
func (l *Lexer) readChar() token.Token {
	startCol := l.col
	open := l.src[l.pos]
	close := rune('\'')
	if open == '‘' {
		close = '’'
	}
	l.advance()
	var value rune
	count := 0
	for l.pos < len(l.src) {
		r := l.src[l.pos]
		if r == close {
			l.advance()
			if count != 1 {
				return token.Token{Type: token.ILLEGAL, Literal: "字符字面量必须包含恰好一个字符", Line: l.line, Column: startCol}
			}
			return token.Token{Type: token.CHAR, Literal: string(value), Line: l.line, Column: startCol}
		}
		if r == '\n' {
			return token.Token{Type: token.ILLEGAL, Literal: "字符字面量未闭合", Line: l.line, Column: startCol}
		}
		if r == '\\' {
			next := l.peek(1)
			escaped, ok := escapeRune(next)
			if !ok {
				return token.Token{Type: token.ILLEGAL, Literal: fmt.Sprintf("未知转义 \\%s", string(next)), Line: l.line, Column: l.col}
			}
			value = escaped
			l.advance()
			l.advance()
		} else {
			value = r
			l.advance()
		}
		count++
		if count > 1 {
			return token.Token{Type: token.ILLEGAL, Literal: "字符字面量必须包含恰好一个字符", Line: l.line, Column: startCol}
		}
	}
	return token.Token{Type: token.ILLEGAL, Literal: "字符字面量未闭合", Line: l.line, Column: startCol}
}

func escapeRune(r rune) (rune, bool) {
	switch r {
	case 'n':
		return '\n', true
	case 't':
		return '\t', true
	case 'r':
		return '\r', true
	case '"':
		return '"', true
	case '\'':
		return '\'', true
	case '\\':
		return '\\', true
	}
	return 0, false
}

func (l *Lexer) readIdentifierOrKeyword() token.Token {
	start := l.pos
	startCol := l.col
	for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
		l.advance()
	}
	literal := string(l.src[start:l.pos])
	if l.prevTok == token.RPAREN && strings.HasPrefix(literal, "的") &&
		len([]rune(literal)) > 1 {
		return l.splitMemberAfterParen(start, startCol, !l.spaceBefore)
	}

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
		return token.Token{Type: t, Literal: literal, Line: l.line, Column: startCol, LeadingSpace: l.spaceBefore}
	}
	return token.Token{Type: token.IDENT, Literal: literal, Line: l.line, Column: startCol, LeadingSpace: l.spaceBefore}
}

// splitMemberAfterParen 处理右括号后的“的成员”粘连写法。
// noSpace 表示右括号与“的”之间没有空白（属于同一条多段调用）。
func (l *Lexer) splitMemberAfterParen(start int, startCol int, noSpace bool) token.Token {
	end := start + len([]rune("的"))
	if end > len(l.src) || (end < len(l.src) && l.src[end] == '\n') {
		return token.Token{Type: token.IDENT, Literal: string(l.src[start:l.pos]), Line: l.line, Column: startCol}
	}
	l.pos = end
	return token.Token{Type: token.MEMBER, Literal: "的", Line: l.line, Column: startCol, LeadingSpace: !noSpace}
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
