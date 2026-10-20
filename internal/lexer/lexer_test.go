package lexer

import (
	"testing"

	"fan/internal/token"
)

func collect(t *testing.T, src string) []token.Token {
	t.Helper()
	l := New(src)
	var out []token.Token
	for {
		tok := l.NextToken()
		out = append(out, tok)
		if tok.Type == token.EOF {
			return out
		}
	}
}

func expect(t *testing.T, src string, want []token.Token) {
	t.Helper()
	got := collect(t, src)
	if len(got) != len(want) {
		t.Fatalf("token 数量不符：期望 %d，实际 %d，源码 %q\n实际：%+v", len(want), len(got), src, got)
	}
	for i, w := range want {
		g := got[i]
		if g.Type != w.Type || g.Literal != w.Literal {
			t.Errorf("第 %d 个 token 不符：期望 {%s,%q}，实际 {%s,%q}", i, w.Type, w.Literal, g.Type, g.Literal)
		}
	}
}

func TestBasicSymbols(t *testing.T) {
	expect(t, "+ - * / % ( ) = == != < <= > >= && || !", []token.Token{
		{Type: token.PLUS, Literal: "+"},
		{Type: token.MINUS, Literal: "-"},
		{Type: token.STAR, Literal: "*"},
		{Type: token.SLASH, Literal: "/"},
		{Type: token.PERCENT, Literal: "%"},
		{Type: token.LPAREN, Literal: "("},
		{Type: token.RPAREN, Literal: ")"},
		{Type: token.ASSIGN, Literal: "="},
		{Type: token.EQ, Literal: "=="},
		{Type: token.NEQ, Literal: "!="},
		{Type: token.LT, Literal: "<"},
		{Type: token.LTE, Literal: "<="},
		{Type: token.GT, Literal: ">"},
		{Type: token.GTE, Literal: ">="},
		{Type: token.AND, Literal: "&&"},
		{Type: token.OR, Literal: "||"},
		{Type: token.NOT, Literal: "!"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestQuestionAndColon(t *testing.T) {
	expect(t, "? ？ :", []token.Token{
		{Type: token.QUEST, Literal: "?"},
		{Type: token.QUEST, Literal: "？"},
		{Type: token.COLON, Literal: ":"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestCompoundAndUpdateSymbols(t *testing.T) {
	expect(t, "+= -= *= /= ++ --", []token.Token{
		{Type: token.PLUS_EQ, Literal: "+="},
		{Type: token.MINUS_EQ, Literal: "-="},
		{Type: token.STAR_EQ, Literal: "*="},
		{Type: token.SLASH_EQ, Literal: "/="},
		{Type: token.INC, Literal: "++"},
		{Type: token.DEC, Literal: "--"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestChineseKeywords(t *testing.T) {
	expect(t, "定义 变量 常量 整数 小数 字符串 布尔 真 假 空 为 加 减 乘 除 取余 等于 不等于 小于 小于等于 大于 大于等于 且 或 非", []token.Token{
		{Type: token.DEFINE, Literal: "定义"},
		{Type: token.VAR, Literal: "变量"},
		{Type: token.CONST, Literal: "常量"},
		{Type: token.TYPE_INT, Literal: "整数"},
		{Type: token.TYPE_FLOAT, Literal: "小数"},
		{Type: token.TYPE_STRING, Literal: "字符串"},
		{Type: token.TYPE_BOOL, Literal: "布尔"},
		{Type: token.TRUE, Literal: "真"},
		{Type: token.FALSE, Literal: "假"},
		{Type: token.NIL, Literal: "空"},
		{Type: token.ASSIGN, Literal: "为"},
		{Type: token.PLUS, Literal: "加"},
		{Type: token.MINUS, Literal: "减"},
		{Type: token.STAR, Literal: "乘"},
		{Type: token.SLASH, Literal: "除"},
		{Type: token.PERCENT, Literal: "取余"},
		{Type: token.EQ, Literal: "等于"},
		{Type: token.NEQ, Literal: "不等于"},
		{Type: token.LT, Literal: "小于"},
		{Type: token.LTE, Literal: "小于等于"},
		{Type: token.GT, Literal: "大于"},
		{Type: token.GTE, Literal: "大于等于"},
		{Type: token.AND, Literal: "且"},
		{Type: token.OR, Literal: "或"},
		{Type: token.NOT, Literal: "非"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestIdentifiersChinese(t *testing.T) {
	expect(t, "名字 年龄2 _临时 abc", []token.Token{
		{Type: token.IDENT, Literal: "名字"},
		{Type: token.IDENT, Literal: "年龄2"},
		{Type: token.IDENT, Literal: "_临时"},
		{Type: token.IDENT, Literal: "abc"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestNumbers(t *testing.T) {
	expect(t, "0 42 3.14 100", []token.Token{
		{Type: token.INT, Literal: "0"},
		{Type: token.INT, Literal: "42"},
		{Type: token.FLOAT, Literal: "3.14"},
		{Type: token.INT, Literal: "100"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestString(t *testing.T) {
	expect(t, `"你好" "含\n换行" "含\"引号\""`, []token.Token{
		{Type: token.STRING, Literal: "你好"},
		{Type: token.STRING, Literal: "含\n换行"},
		{Type: token.STRING, Literal: `含"引号"`},
		{Type: token.EOF, Literal: ""},
	})
}

func TestStringUnterminated(t *testing.T) {
	got := collect(t, `"未闭合`)
	if got[0].Type != token.ILLEGAL {
		t.Fatalf("未闭合字符串应返回 ILLEGAL，得到 %+v", got[0])
	}
}

func TestStringInterpolationIdent(t *testing.T) {
	expect(t, `"你好 $名称"`, []token.Token{
		{Type: token.STRING_PART, Literal: "你好 "},
		{Type: token.INTERP_IDENT, Literal: "名称"},
		{Type: token.INTERP_END, Literal: ""},
		{Type: token.EOF, Literal: ""},
	})
}

func TestStringInterpolationExpr(t *testing.T) {
	expect(t, `"a${x 加 1}b"`, []token.Token{
		{Type: token.STRING_PART, Literal: "a"},
		{Type: token.INTERP_START, Literal: ""},
		{Type: token.IDENT, Literal: "x"},
		{Type: token.PLUS, Literal: "加"},
		{Type: token.INT, Literal: "1"},
		{Type: token.INTERP_END, Literal: ""},
		{Type: token.STRING_PART, Literal: "b"},
		{Type: token.INTERP_END, Literal: ""},
		{Type: token.EOF, Literal: ""},
	})
}

func TestStringDollarEscapedAndLiteral(t *testing.T) {
	expect(t, `"\$x $100"`, []token.Token{
		{Type: token.STRING, Literal: "$x $100"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestEllipsisAndDots(t *testing.T) {
	expect(t, "... a.b ...x", []token.Token{
		{Type: token.ELLIPSIS, Literal: "..."},
		{Type: token.IDENT, Literal: "a"},
		{Type: token.DOT, Literal: "."},
		{Type: token.IDENT, Literal: "b"},
		{Type: token.ELLIPSIS, Literal: "..."},
		{Type: token.IDENT, Literal: "x"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestHashComment(t *testing.T) {
	expect(t, "# 这是注释\n年龄 = 18 # 行尾", []token.Token{
		{Type: token.NEWLINE, Literal: "\n"},
		{Type: token.IDENT, Literal: "年龄"},
		{Type: token.ASSIGN, Literal: "="},
		{Type: token.INT, Literal: "18"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestChineseCommentWithSpace(t *testing.T) {
	expect(t, "注释 这是注释\n年龄 = 18", []token.Token{
		{Type: token.NEWLINE, Literal: "\n"},
		{Type: token.IDENT, Literal: "年龄"},
		{Type: token.ASSIGN, Literal: "="},
		{Type: token.INT, Literal: "18"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestChineseCommentWithColon(t *testing.T) {
	expect(t, "注释:内容\n注释：内容\n年龄 = 18", []token.Token{
		{Type: token.NEWLINE, Literal: "\n"},
		{Type: token.NEWLINE, Literal: "\n"},
		{Type: token.IDENT, Literal: "年龄"},
		{Type: token.ASSIGN, Literal: "="},
		{Type: token.INT, Literal: "18"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestChineseCommentEndOfLine(t *testing.T) {
	expect(t, "注释\n年龄", []token.Token{
		{Type: token.NEWLINE, Literal: "\n"},
		{Type: token.IDENT, Literal: "年龄"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestChineseCommentIdentifierWithoutSeparator(t *testing.T) {
	expect(t, "注释abc = 1", []token.Token{
		{Type: token.IDENT, Literal: "注释abc"},
		{Type: token.ASSIGN, Literal: "="},
		{Type: token.INT, Literal: "1"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestStringContainsCommentTokens(t *testing.T) {
	expect(t, `"价格#100" "含 注释 字样"`, []token.Token{
		{Type: token.STRING, Literal: "价格#100"},
		{Type: token.STRING, Literal: "含 注释 字样"},
		{Type: token.EOF, Literal: ""},
	})
}

func TestNewlineAndPosition(t *testing.T) {
	l := New("变量 名字\n年龄")
	first := l.NextToken()
	if first.Type != token.VAR || first.Line != 1 {
		t.Fatalf("首 token 应在第 1 行，得到 %+v", first)
	}
	l.NextToken()
	newline := l.NextToken()
	if newline.Type != token.NEWLINE {
		t.Fatalf("应为换行 token，得到 %+v", newline)
	}
	third := l.NextToken()
	if third.Type != token.IDENT || third.Line != 2 {
		t.Fatalf("换行后 token 应在第 2 行，得到 %+v", third)
	}
}

func TestVarDeclarationTokens(t *testing.T) {
	expect(t, "定义 变量 整数 年龄 为 18", []token.Token{
		{Type: token.DEFINE, Literal: "定义"},
		{Type: token.VAR, Literal: "变量"},
		{Type: token.TYPE_INT, Literal: "整数"},
		{Type: token.IDENT, Literal: "年龄"},
		{Type: token.ASSIGN, Literal: "为"},
		{Type: token.INT, Literal: "18"},
		{Type: token.EOF, Literal: ""},
	})
}
