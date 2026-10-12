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

func TestMixfixEmptyLastSegment(t *testing.T) {
	src := `函数 修改(值 字符串)的状态为无效() -> 字符串
    返回 值 + ":无效"
结束

修改("a")的状态为无效()`
	if got := evalPrimitiveMethodSource(t, src).Inspect(); got != "a:无效" {
		t.Fatalf("带括号末段错误：%s", got)
	}

	src2 := `函数 修改(值 字符串)的状态为无效 -> 字符串
    返回 值 + ":无效"
结束

修改("b")的状态为无效`
	if got := evalPrimitiveMethodSource(t, src2).Inspect(); got != "b:无效" {
		t.Fatalf("省略括号末段错误：%s", got)
	}
}

func TestMixfixSpacedChainPrefersNormalFunction(t *testing.T) {
	src := `模型 结果
    变量 字符串 标签
结束

函数 修改(值 字符串)
    变量 r = 结果()
    r.标签 = 值 + ":标签"
    返回 r
结束

函数 标记(值 字符串)的状态为无效 -> 字符串
    返回 "不应走到分段"
结束

变量 x = 修改("z") 的标签
x`
	prog, errs := parser.ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	res, err := Eval(prog, NewEnvironment())
	if err != nil {
		t.Fatalf("运行失败：%v", err)
	}
	if res.Inspect() != "z:标签" {
		t.Fatalf("有空格时应先走普通函数返回值链，实际：%s", res.Inspect())
	}
}
