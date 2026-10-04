package parser

import (
	"testing"

	"fan/internal/ast"
)

func TestParseFunctionTag(t *testing.T) {
	src := `@内建("测试.加一")
定义 函数 加一(n 整数) -> 整数
    返回 空
结束`
	prog, errs := ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析错误：%v", errs)
	}
	decl := prog.Statements[0].(*ast.VarDecl)
	fn := decl.Value.(*ast.FunctionLiteral)
	if len(fn.Tags) != 1 {
		t.Fatalf("应解析 1 个标签，实际 %d", len(fn.Tags))
	}
	tag := fn.Tags[0]
	if tag.TypeName != "内建" {
		t.Fatalf("标签名称错误：%s", tag.TypeName)
	}
	if len(tag.Positional) != 1 {
		t.Fatalf("标签位置参数数量错误：%d", len(tag.Positional))
	}
}

func TestParseClassTag(t *testing.T) {
	src := `@标记(说明 = "模型")
模型 用户
    变量 名称
结束`
	prog, errs := ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析错误：%v", errs)
	}
	cls := prog.Statements[0].(*ast.ClassStmt)
	if len(cls.Tags) != 1 {
		t.Fatalf("应解析 1 个模型标签，实际 %d", len(cls.Tags))
	}
	if cls.Tags[0].TypeName != "标记" {
		t.Fatalf("标签名称错误：%s", cls.Tags[0].TypeName)
	}
}

func TestParseMethodTag(t *testing.T) {
	src := `模型 用户
结束
@内建("打印")
定义 用户 的 方法 输出()
结束`
	prog, errs := ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析错误：%v", errs)
	}
	method := prog.Statements[1].(*ast.MethodDef)
	if len(method.Function.Tags) != 1 {
		t.Fatalf("应解析 1 个方法标签，实际 %d", len(method.Function.Tags))
	}
}

func TestParseShorthandMemberCallInArgument(t *testing.T) {
	src := `打印(r 的 面积)`
	prog, errs := ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析错误：%v", errs)
	}
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.CallExpr)
	if _, ok := call.Args[0].(*ast.MaybeCallExpr); !ok {
		t.Fatal("参数中的成员表达式应自动无参调用")
	}
}

func TestParseBareIdentifierArgumentStaysFunction(t *testing.T) {
	src := `注册标签处理("测试", 处理函数)`
	prog, errs := ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析错误：%v", errs)
	}
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.CallExpr)
	if _, ok := call.Args[1].(*ast.Identifier); !ok {
		t.Fatal("参数中的裸标识符应作为函数值传递")
	}
}

func TestParseTaggedExport(t *testing.T) {
	src := `@内建("type")
导出 函数 type(值) -> 字符串
结束`
	prog, errs := ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析错误：%v", errs)
	}
	exportStmt := prog.Statements[0].(*ast.ExportStmt)
	decl := exportStmt.Inner.(*ast.VarDecl)
	fn := decl.Value.(*ast.FunctionLiteral)
	if len(fn.Tags) != 1 {
		t.Fatalf("应解析 1 个标签，实际 %d", len(fn.Tags))
	}
}

func TestParseInvalidTagTarget(t *testing.T) {
	src := `@标记
变量 x = 1`
	_, errs := ParseProgram(src)
	if len(errs) == 0 {
		t.Fatal("标签用于非函数变量应报错")
	}
}

func TestParseNamedTagArg(t *testing.T) {
	src := `@路由信息("/用户", 方法 = "POST")
定义 函数 创建用户()
    返回 空
结束`
	prog, errs := ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析错误：%v", errs)
	}
	decl := prog.Statements[0].(*ast.VarDecl)
	fn := decl.Value.(*ast.FunctionLiteral)
	tag := fn.Tags[0]
	if len(tag.Positional) != 1 || len(tag.Named) != 1 {
		t.Fatalf("标签参数数量错误：%d %d", len(tag.Positional), len(tag.Named))
	}
	if tag.Named[0].Name != "方法" {
		t.Fatalf("命名参数名称错误：%s", tag.Named[0].Name)
	}
}
