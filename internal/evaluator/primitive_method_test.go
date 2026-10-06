package evaluator

import (
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func evalPrimitiveMethodSource(t *testing.T, src string) object.Object {
	t.Helper()
	prog, errs := parser.ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	var program *ast.Program = prog
	env := NewEnvironment()
	res, err := Eval(program, env)
	if err != nil {
		t.Fatalf("运行失败：%v", err)
	}
	return res
}

func TestPrimitiveMethodDefinitionAndCall(t *testing.T) {
	src := `定义 字符串 的 方法 添加括号() -> 字符串
    返回 "[" + 自己 + "]"
结束

"abc".添加括号()`
	if got := evalPrimitiveMethodSource(t, src).Inspect(); got != "[abc]" {
		t.Fatalf("结果为 %s，期望 [abc]", got)
	}
}

func TestPrimitiveMethodExplicitReceiver(t *testing.T) {
	src := `定义 文本 字符串 的 方法 重复两次() -> 字符串
    返回 文本 + 文本
结束

"x".重复两次()`
	if got := evalPrimitiveMethodSource(t, src).Inspect(); got != "xx" {
		t.Fatalf("结果为 %s，期望 xx", got)
	}
}

func TestPrimitiveMethodOverride(t *testing.T) {
	src := `定义 字符串 的 方法 测试覆盖标记() -> 字符串
    返回 "旧"
结束
定义 字符串 的 方法 测试覆盖标记() -> 字符串
    返回 "新"
结束

"x".测试覆盖标记()`
	if got := evalPrimitiveMethodSource(t, src).Inspect(); got != "新" {
		t.Fatalf("结果为 %s，期望 新", got)
	}
}

func TestPrimitiveMethodWithArgs(t *testing.T) {
	src := `定义 字符串 的 方法 包围(左 字符串, 右 字符串) -> 字符串
    返回 左 + 自己 + 右
结束

"x".包围("(", ")")`
	if got := evalPrimitiveMethodSource(t, src).Inspect(); got != "(x)" {
		t.Fatalf("结果为 %s，期望 (x)", got)
	}
}

func TestPrimitiveMethodOnlyStringCurrently(t *testing.T) {
	src := `定义 整数 的 方法 标记() -> 字符串
    返回 "x"
结束`
	prog, errs := parser.ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	var program *ast.Program = prog
	if _, err := Eval(program, NewEnvironment()); err == nil {
		t.Fatal("整数方法当前应报错")
	}
}

func TestPrimitiveMethodIsolation(t *testing.T) {
	definition := `定义 字符串 的 方法 隔离方法() -> 字符串
    返回 "存在"
结束`
	prog, errs := parser.ParseProgram(definition)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	var program *ast.Program = prog
	first := NewEnvironment()
	if _, err := Eval(program, first); err != nil {
		t.Fatalf("运行失败：%v", err)
	}
	second := NewEnvironment()
	callProg, errs := parser.ParseProgram(`"x".隔离方法()`)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	var callProgram *ast.Program = callProg
	if _, err := Eval(callProgram, second); err == nil {
		t.Fatal("方法泄漏到新环境")
	}
}

func TestPrimitiveMethodNoArgumentAutoCall(t *testing.T) {
	src := `定义 字符串 的 方法 加星号() -> 字符串
    返回 自己 + "*"
结束

"x".加星号`
	if got := evalPrimitiveMethodSource(t, src).Inspect(); got != "x*" {
		t.Fatalf("结果为 %s，期望 x*", got)
	}
}

func TestPrimitiveMethodFunctionEquivalence(t *testing.T) {
	src := `定义 函数 包围文本(自己 字符串, 记号 字符串) -> 字符串
    返回 记号 + 自己 + 记号
结束
定义 字符串 的 方法 包围文本(记号 字符串) -> 字符串
    返回 记号 + 自己 + 记号
结束

包围文本("x", "#")`
	if got := evalPrimitiveMethodSource(t, src).Inspect(); got != "#x#" {
		t.Fatalf("结果为 %s，期望 #x#", got)
	}
}
