package parser

import (
	"strings"
	"testing"

	"fan/internal/ast"
)

func TestClassBasic(t *testing.T) {
	src := strings.Join([]string{
		`模型 矩形`,
		`    变量 整数 宽`,
		`    变量 整数 高`,
		`结束`,
	}, "\n")
	prog := mustParse(t, src)
	cls := prog.Statements[0].(*ast.ClassStmt)
	if cls.Name != "矩形" {
		t.Fatalf("模型名应为 矩形，实际 %s", cls.Name)
	}
	if len(cls.Fields) != 2 {
		t.Fatalf("应有 2 个字段，实际 %d", len(cls.Fields))
	}
	if cls.Fields[0].Type != ast.TypeInt {
		t.Fatalf("字段类型错误：%s", cls.Fields[0].Type)
	}
}

func TestClassEmbed(t *testing.T) {
	src := strings.Join([]string{
		`模型 学生`,
		`    嵌入 人`,
		`    变量 整数 学号`,
		`结束`,
	}, "\n")
	prog := mustParse(t, src)
	cls := prog.Statements[0].(*ast.ClassStmt)
	if len(cls.Embeds) != 1 || cls.Embeds[0] != "人" {
		t.Fatalf("嵌入解析错误：%v", cls.Embeds)
	}
}

func TestClassMissingName(t *testing.T) {
	expectError(t, "模型\n    变量 x\n结束")
}

func TestClassMissingEnd(t *testing.T) {
	expectError(t, "模型 矩形\n    变量 宽")
}

func TestClassNoMethodsInside(t *testing.T) {
	src := "模型 矩形\n    函数 面积(自身)\n        返回 1\n    结束\n结束"
	expectError(t, src)
}

func TestExternalMethodDef(t *testing.T) {
	src := strings.Join([]string{
		`模型 矩形`,
		`    变量 整数 宽`,
		`    变量 整数 高`,
		`结束`,
		`定义 矩形 的 方法 面积（r 矩形） -> 整数`,
		`    返回 自己.宽 乘 自己.高`,
		`结束`,
	}, "\n")
	prog := mustParse(t, src)
	method, ok := prog.Statements[1].(*ast.MethodDef)
	if !ok {
		t.Fatalf("应为方法定义，实际 %T", prog.Statements[1])
	}
	if method.ClassName != "矩形" || method.MethodName != "面积" {
		t.Fatalf("方法定义解析错误：%+v", method)
	}
	if len(method.Function.ReturnTypes) != 1 || method.Function.ReturnTypes[0] != ast.TypeInt {
		t.Fatalf("返回类型错误：%+v", method.Function.ReturnTypes)
	}
}

func TestExternalMethodWithParams(t *testing.T) {
	src := "定义 矩形 的 方法 放大（倍数 整数）\n    自己.宽 = 自己.宽 乘 倍数\n结束"
	prog := mustParse(t, src)
	method := prog.Statements[0].(*ast.MethodDef)
	if len(method.Function.Params) != 1 {
		t.Fatalf("应有 1 个参数（倍数），实际 %d", len(method.Function.Params))
	}
}

func TestFieldAssign(t *testing.T) {
	prog := mustParse(t, "r.宽 = 5")
	stmt := prog.Statements[0].(*ast.FieldAssignExpr)
	if stmt.Name != "宽" {
		t.Fatalf("字段名应为 宽，实际 %s", stmt.Name)
	}
}

func TestModelKeyword(t *testing.T) {
	src := "模型 矩形\n    变量 整数 宽\n结束"
	prog := mustParse(t, src)
	cls := prog.Statements[0].(*ast.ClassStmt)
	if cls.Name != "矩形" || cls.Fields[0].Type != ast.TypeInt {
		t.Fatalf("模型解析错误：%+v", cls)
	}
}

func TestDefineModelOptional(t *testing.T) {
	src := "定义 模型 矩形\n    变量 宽\n结束"
	prog := mustParse(t, src)
	if _, ok := prog.Statements[0].(*ast.ClassStmt); !ok {
		t.Fatalf("定义 模型 应解析为模型定义，实际 %T", prog.Statements[0])
	}
}

func TestTypedField(t *testing.T) {
	src := "模型 矩形\n    变量 整数 宽\n    变量 整数 高\n结束"
	prog := mustParse(t, src)
	cls := prog.Statements[0].(*ast.ClassStmt)
	if cls.Fields[0].Type != ast.TypeInt || cls.Fields[1].Type != ast.TypeInt {
		t.Fatalf("字段类型错误：%+v", cls.Fields)
	}
}

func TestClassKeywordAliasRemoved(t *testing.T) {
	expectError(t, "类 矩形\n    变量 整数 宽\n结束")
	expectError(t, "定义 类 矩形\n    变量 整数 宽\n结束")
}
