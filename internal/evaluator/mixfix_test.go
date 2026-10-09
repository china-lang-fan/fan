package evaluator

import (
	"testing"

	"fan/internal/ast"
	"fan/internal/parser"
)

func TestMixfixFunctionCall(t *testing.T) {
	src := `函数 修改(值 字符串)的状态为(状态 字符串) -> 字符串
    返回 值 + ":" + 状态
结束

修改("x")的状态为("无效")`
	if got := evalPrimitiveMethodSource(t, src).Inspect(); got != "x:无效" {
		t.Fatalf("结果为 %s，期望 x:无效", got)
	}
}

func TestMixfixFunctionOverloadByRuntimeType(t *testing.T) {
	src := `函数 修改(值 字符串)的状态为(状态 字符串) -> 字符串
    返回 "字符串"
结束
函数 修改(值 整数)的状态为(状态 字符串) -> 字符串
    返回 "整数"
结束

修改(1)的状态为("无效")`
	if got := evalPrimitiveMethodSource(t, src).Inspect(); got != "整数" {
		t.Fatalf("结果为 %s，期望 整数", got)
	}
}

func TestMixfixThreeSegmentFunction(t *testing.T) {
	src := `函数 从(起点 字符串)到(终点 字符串)乘坐(工具 字符串) -> 字符串
    返回 起点 + "->" + 终点 + ":" + 工具
结束

从("北京")到("上海")乘坐("高铁")`
	if got := evalPrimitiveMethodSource(t, src).Inspect(); got != "北京->上海:高铁" {
		t.Fatalf("结果为 %s", got)
	}
}

func TestMixfixNoMatchingSignature(t *testing.T) {
	src := `函数 修改(值 字符串)的状态为(状态 字符串) -> 字符串
    返回 值
结束

修改(1)的状态为("无效")`
	prog, errs := parser.ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	var program *ast.Program = prog
	if _, err := Eval(program, NewEnvironment()); err == nil {
		t.Fatal("不匹配签名应报错")
	}
}
