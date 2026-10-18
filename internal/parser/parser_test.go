package parser

import (
	"testing"

	"fan/internal/ast"
)

func mustParse(t *testing.T, src string) *ast.Program {
	t.Helper()
	prog, errs := ParseProgram(src)
	if len(errs) != 0 {
		t.Fatalf("解析失败：%v（源码 %q）", errs, src)
	}
	return prog
}

func expectError(t *testing.T, src string) {
	t.Helper()
	_, errs := ParseProgram(src)
	if len(errs) == 0 {
		t.Fatalf("期望解析报错但成功：%q", src)
	}
}

func TestVarDeclForms(t *testing.T) {
	cases := []string{
		`名字 为 "小明"`,
		`名字 = "小明"`,
		`定义 名字 为 "小明"`,
		`变量 名字 = "小明"`,
		`定义 变量 名字 为 "小明"`,
		`变量 整数 年龄 = 18`,
		`定义 常量 小数 圆周率 为 3.14`,
		`常量 圆周率 = 3.14`,
	}
	for _, src := range cases {
		prog := mustParse(t, src)
		if len(prog.Statements) != 1 {
			t.Fatalf("应为 1 条语句：%q", src)
		}
		if _, ok := prog.Statements[0].(*ast.VarDecl); !ok {
			t.Fatalf("应为变量声明：%q，实际 %T", src, prog.Statements[0])
		}
	}
}

func TestVarDeclTypeAndConst(t *testing.T) {
	prog := mustParse(t, `定义 常量 小数 圆周率 为 3.14`)
	decl := prog.Statements[0].(*ast.VarDecl)
	if !decl.IsConst {
		t.Fatal("应为常量")
	}
	if decl.DeclType != ast.TypeFloat {
		t.Fatalf("类型应为小数，实际 %q", decl.DeclType)
	}
	if decl.Name != "圆周率" {
		t.Fatalf("名字不符：%s", decl.Name)
	}
}

func TestVarDeclMustHaveValue(t *testing.T) {
	expectError(t, `变量 名字`)
	expectError(t, `定义 整数 年龄`)
	expectError(t, `常量 圆周率`)
}

func TestTernaryMissingColon(t *testing.T) {
	expectError(t, "真 ? 1")
}

func TestArithmeticPrecedence(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"1 + 2 * 3", "(1 + (2 * 3))"},
		{"1 * 2 + 3", "((1 * 2) + 3)"},
		{"10 / 2 % 3", "((10 / 2) % 3)"},
		{"-1 + 2", "((-1) + 2)"},
		{"1 + 2 == 3", "((1 + 2) == 3)"},
		{"真 且 假 或 真", "((真 且 假) 或 真)"},
		{"非 真 且 假", "((非 真) 且 假)"},
		{"1 加 2 乘 3", "(1 加 (2 乘 3))"},
		{"10 除 2 取余 3", "((10 除 2) 取余 3)"},
		{"1 小于 2 等于 真", "((1 小于 2) 等于 真)"},
		{"(1 + 2) * 3", "((1 + 2) * 3)"},
		{"1 + 2 + 3 + 4", "(((1 + 2) + 3) + 4)"},
		{"真 ? 1 : 2", "(真 ? 1 : 2)"},
		{"真 ? 1 + 2 : 3", "(真 ? (1 + 2) : 3)"},
		{"假 ? 1 : 真 ? 2 : 3", "(假 ? 1 : (真 ? 2 : 3))"},
	}
	for _, c := range cases {
		prog := mustParse(t, c.src)
		stmt := prog.Statements[0].(*ast.ExpressionStmt)
		if got := stmt.Expression.String(); got != c.want {
			t.Errorf("源码 %q：期望 %s，实际 %s", c.src, c.want, got)
		}
	}
}

func TestLiterals(t *testing.T) {
	prog := mustParse(t, "42\n3.14\n\"你好\"\n真\n假\n空\n")
	if len(prog.Statements) != 6 {
		t.Fatalf("应有 6 条语句，实际 %d", len(prog.Statements))
	}
}

func TestBareReassignment(t *testing.T) {
	prog := mustParse(t, "年龄 = 20")
	decl := prog.Statements[0].(*ast.VarDecl)
	if decl.Name != "年龄" {
		t.Fatalf("名字不符：%s", decl.Name)
	}
}

func TestBlankLinesAndComments(t *testing.T) {
	src := "# 开头注释\n\n定义 变量 甲 = 1\n注释：行间注释\n乙 = 甲 加 1 # 行尾\n"
	prog := mustParse(t, src)
	if len(prog.Statements) != 2 {
		t.Fatalf("应有 2 条语句，实际 %d", len(prog.Statements))
	}
}

func TestStringEscapes(t *testing.T) {
	prog := mustParse(t, `值 = "a\nb\"c\\"`)
	decl := prog.Statements[0].(*ast.VarDecl)
	lit := decl.Value.(*ast.StringLiteral)
	want := "a\nb\"c\\"
	if lit.Value != want {
		t.Fatalf("期望 %q，实际 %q", want, lit.Value)
	}
}

func TestStringInterpolationAST(t *testing.T) {
	prog := mustParse(t, `变量 名称 = "凡"
变量 s = "你好 $名称 版本 ${名称 加 1}!"`)
	decl := prog.Statements[1].(*ast.VarDecl)
	node, ok := decl.Value.(*ast.InterpString)
	if !ok {
		t.Fatalf("期望 InterpString，得到 %T", decl.Value)
	}
	if len(node.Exprs) != 2 {
		t.Fatalf("期望 2 个插值表达式，得到 %d", len(node.Exprs))
	}
	if id, ok := node.Exprs[0].(*ast.Identifier); !ok || id.Name != "名称" {
		t.Fatalf("第一个插值应为标识符 名称，得到 %T", node.Exprs[0])
	}
	wantParts := []string{"你好 ", " 版本 ", "!"}
	if len(node.Parts) != len(wantParts) {
		t.Fatalf("片段数期望 %d，得到 %d：%q", len(wantParts), len(node.Parts), node.Parts)
	}
	for i, w := range wantParts {
		if node.Parts[i] != w {
			t.Fatalf("片段 %d 期望 %q，得到 %q", i, w, node.Parts[i])
		}
	}
}

func TestStringInterpolationAdjacent(t *testing.T) {
	prog := mustParse(t, `"$a$b"`)
	lit := prog.Statements[0].(*ast.ExpressionStmt).Expression.(*ast.InterpString)
	if len(lit.Exprs) != 2 {
		t.Fatalf("相邻插值应有 2 个表达式，得到 %d", len(lit.Exprs))
	}
	if len(lit.Parts) != 3 {
		t.Fatalf("相邻插值应有 3 个文本片段（均可为空），得到 %d %q", len(lit.Parts), lit.Parts)
	}
}

func TestParenthesesAcrossLines(t *testing.T) {
	prog := mustParse(t, "(1 +\n 2)")
	if len(prog.Statements) != 1 {
		t.Fatalf("括号内折行应合法，实际语句数 %d", len(prog.Statements))
	}
}
