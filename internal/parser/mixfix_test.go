package parser

import (
	"testing"

	"fan/internal/ast"
)

func TestMixfixFunctionDefinition(t *testing.T) {
	src := "函数 修改用户(当前用户 用户)的状态为(新状态 状态) -> 字符串\n    返回 当前用户.名 + 新状态.值\n结束"
	prog := mustParse(t, src)
	decl := prog.Statements[0].(*ast.VarDecl)
	if decl.Name != "修改用户的状态为" {
		t.Fatalf("全名错误：%s", decl.Name)
	}
	literal := decl.Value.(*ast.FunctionLiteral)
	if len(literal.NameSegments) != 2 || literal.NameSegments[0] != "修改用户" || literal.NameSegments[1] != "状态为" {
		t.Fatalf("分段名错误：%v", literal.NameSegments)
	}
	if literal.Signature != "修改用户(用户)的状态为(状态)" {
		t.Fatalf("签名错误：%s", literal.Signature)
	}
	if len(literal.Params) != 2 {
		t.Fatalf("参数数量错误：%d", len(literal.Params))
	}
}

func TestMixfixFunctionRequiresExplicitTypes(t *testing.T) {
	expectError(t, "函数 标记(值)的状态(新值 字符串)\n结束")
	expectError(t, "函数 标记(值 字符串)的状态()\n结束")
	expectError(t, "函数 标记(值 字符串 = \"x\")的状态(新值 字符串)\n结束")
	expectError(t, "函数 标记(值 ...字符串)的状态(新值 字符串)\n结束")
}
