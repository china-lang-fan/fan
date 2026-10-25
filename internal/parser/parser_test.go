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

func TestVarDeclInitialValue(t *testing.T) {
	expectError(t, `常量 圆周率`)

	prog := mustParse(t, `变量 名字`)
	if len(prog.Statements) != 1 {
		t.Fatalf("应为 1 条语句")
	}
	vd, ok := prog.Statements[0].(*ast.VarDecl)
	if !ok || vd.Value != nil {
		t.Fatalf("变量 名字 应为无初值声明")
	}

	prog = mustParse(t, `定义 整数 年龄`)
	vd = prog.Statements[0].(*ast.VarDecl)
	if vd.Value != nil || vd.DeclType != ast.TypeInt {
		t.Fatalf("定义 整数 年龄 应为整数类型无初值声明")
	}
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

func TestCompositeLiteralAST(t *testing.T) {
	prog := mustParse(t, `变量 r = 矩形{宽: 3, 高: 4}`)
	decl := prog.Statements[0].(*ast.VarDecl)
	lit, ok := decl.Value.(*ast.CompositeLiteral)
	if !ok {
		t.Fatalf("期望 CompositeLiteral，得到 %T", decl.Value)
	}
	model, ok := lit.Model.(*ast.Identifier)
	if !ok || model.Name != "矩形" {
		t.Fatalf("模型名错误：%T %v", lit.Model, lit.Model)
	}
	if len(lit.Fields) != 2 || lit.Fields[0].Name != "宽" || lit.Fields[1].Name != "高" {
		t.Fatalf("字段解析错误：%+v", lit.Fields)
	}
}

func TestCompositeLiteralNestedAndTrailingComma(t *testing.T) {
	prog := mustParse(t, "变量 p = 人{姓名: \"甲\", 宠物: 狗{名字: \"乙\"},}")
	decl := prog.Statements[0].(*ast.VarDecl)
	lit := decl.Value.(*ast.CompositeLiteral)
	if len(lit.Fields) != 2 {
		t.Fatalf("应有 2 个字段（尾随逗号合法），得到 %d", len(lit.Fields))
	}
	nested, ok := lit.Fields[1].Value.(*ast.CompositeLiteral)
	if !ok {
		t.Fatalf("第二个字段应为嵌套复合字面量，得到 %T", lit.Fields[1].Value)
	}
	if id, _ := nested.Model.(*ast.Identifier); id == nil || id.Name != "狗" {
		t.Fatalf("嵌套模型名错误：%v", nested.Model)
	}
}

func TestCompositeLiteralModuleModel(t *testing.T) {
	prog := mustParse(t, `变量 r = 图形.矩形{宽: 3}`)
	lit := prog.Statements[0].(*ast.VarDecl).Value.(*ast.CompositeLiteral)
	m, ok := lit.Model.(*ast.MemberExpr)
	if !ok || m.Name != "矩形" {
		t.Fatalf("跨模块模型名错误：%T", lit.Model)
	}
}

func TestCompositeLiteralDuplicateFieldFails(t *testing.T) {
	if _, errs := ParseProgram(`变量 r = 矩形{宽: 1, 宽: 2}`); len(errs) == 0 {
		t.Fatal("重复字段应解析报错")
	}
}

func TestSliceExprAST(t *testing.T) {
	cases := []struct {
		src     string
		lowNil  bool
		highNil bool
	}{
		{`数列[1:3]`, false, false},
		{`数列[2:]`, false, true},
		{`数列[:3]`, true, false},
		{`数列[:]`, true, true},
	}
	for _, c := range cases {
		prog := mustParse(t, c.src)
		expr := prog.Statements[0].(*ast.ExpressionStmt).Expression
		sl, ok := expr.(*ast.SliceExpr)
		if !ok {
			t.Fatalf("%s 期望 SliceExpr，得到 %T", c.src, expr)
		}
		if (sl.Low == nil) != c.lowNil {
			t.Errorf("%s Low 省略状态错误", c.src)
		}
		if (sl.High == nil) != c.highNil {
			t.Errorf("%s High 省略状态错误", c.src)
		}
	}
}

func TestSpreadAST(t *testing.T) {
	prog := mustParse(t, `[...前, 1, ...后]`)
	lit := prog.Statements[0].(*ast.ExpressionStmt).Expression.(*ast.ArrayLiteral)
	if len(lit.Elements) != 3 {
		t.Fatalf("应有 3 个元素，得到 %d", len(lit.Elements))
	}
	if _, ok := lit.Elements[0].(*ast.SpreadExpr); !ok {
		t.Fatalf("第一个元素应为 SpreadExpr，得到 %T", lit.Elements[0])
	}
	if _, ok := lit.Elements[2].(*ast.SpreadExpr); !ok {
		t.Fatalf("第三个元素应为 SpreadExpr，得到 %T", lit.Elements[2])
	}
}

func TestDictSpreadAST(t *testing.T) {
	prog := mustParse(t, `{...d, "a": 1}`)
	lit := prog.Statements[0].(*ast.ExpressionStmt).Expression.(*ast.DictLiteral)
	if len(lit.Pairs) != 2 || !lit.Pairs[0].Spread {
		t.Fatalf("字典展开解析错误：%+v", lit.Pairs)
	}
}

func TestParenthesesAcrossLines(t *testing.T) {
	prog := mustParse(t, "(1 +\n 2)")
	if len(prog.Statements) != 1 {
		t.Fatalf("括号内折行应合法，实际语句数 %d", len(prog.Statements))
	}
}

func TestTypeDefForms(t *testing.T) {
	cases := []struct {
		src  string
		base ast.DeclType
	}{
		{`类型 状态 整数`, ast.TypeInt},
		{`定义 类型 状态 = 整数`, ast.TypeInt},
		{`定义 类型 编号 为 整数 或 字符串`, ast.DeclType("整数|字符串")},
		{`定义 类型 编号 = 整数 | 字符串`, ast.DeclType("整数|字符串")},
	}
	for _, c := range cases {
		prog := mustParse(t, c.src)
		if len(prog.Statements) != 1 {
			t.Fatalf("应为 1 条语句：%q", c.src)
		}
		td, ok := prog.Statements[0].(*ast.TypeDefStmt)
		if !ok {
			t.Fatalf("应为类型声明：%q", c.src)
		}
		if td.Name != "状态" && c.src != `类型 状态 整数` {
			if td.Name != "编号" {
				t.Errorf("%s 类型名应为 编号，实际 %s", c.src, td.Name)
			}
		}
		if td.BaseType != c.base {
			t.Errorf("%s 底层类型应为 %s，实际 %s", c.src, c.base, td.BaseType)
		}
	}
	expectError(t, `定义 类型 编号 为`)
	expectError(t, `定义 类型 编号 为 真`)
}

func TestDefineFirstForms(t *testing.T) {
	cases := []string{
		`定义 变量 x = 1`,
		`定义 常量 y = 2`,
		`定义 函数 f() -> 整数
返回 1
结束`,
		`定义 模型 M
变量 整数 a
结束`,
		`定义 接口 I
方法 f() -> 整数
结束`,
		`定义 类型 T 为 整数`,
		`定义 公开 函数 g() -> 整数
返回 1
结束`,
		`定义 公开 模型 N
变量 整数 a
结束`,
		`定义 私有 类型 U = 整数 或 字符串`,
		`公开 定义 函数 h() -> 整数
返回 1
结束`,
	}
	for _, src := range cases {
		prog := mustParse(t, src)
		if len(prog.Statements) != 1 {
			t.Fatalf("应为 1 条语句：%q", src)
		}
	}
}
