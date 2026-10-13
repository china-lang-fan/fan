package evaluator

import (
	"strings"
	"testing"

	"fan/internal/object"
)

func mustErrorValue(t *testing.T, src string) *object.Error {
	t.Helper()
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	e, ok := res.(*object.Error)
	if !ok {
		t.Fatalf("期望错误值，实际 %T：%s", res, res.Inspect())
	}
	return e
}

func TestErrorWithCodeAndDetails(t *testing.T) {
	e := mustErrorValue(t, `错误("余额不足", "PAY_001", {"金额": 10})`)
	if e.Message != "余额不足" {
		t.Fatalf("消息错误：%s", e.Message)
	}
	if e.Code != "PAY_001" {
		t.Fatalf("错误码错误：%s", e.Code)
	}
	if e.Details == nil {
		t.Fatal("详情不应为空")
	}
	v, ok := e.Details.Get(&object.String{Value: "金额"})
	if !ok {
		t.Fatal("详情缺少金额")
	}
	if n, ok := v.(*object.Integer); !ok || n.Value != 10 {
		t.Fatalf("详情金额错误：%v", v)
	}
}

func TestErrorMembers(t *testing.T) {
	res, err := runSource(t, `
变量 e = 错误("失败", "E1")
e.错误码`)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := res.(*object.String); !ok || s.Value != "E1" {
		t.Fatalf("错误码成员错误：%v", res)
	}
}

func TestCheckPropagatesErrorWithStack(t *testing.T) {
	src := `
函数 底层() -> (字符串, 错误)
    返回 "", 错误("打不开", "IO")
结束
函数 中层() -> (字符串, 错误)
    返回 检查 底层()
结束
函数 顶层() -> (字符串, 错误)
    返回 检查 中层()
结束
变量 结果, e = 顶层()
e`
	e := mustErrorValue(t, src)
	if e.Code != "IO" {
		t.Fatalf("错误码应传播：%s", e.Code)
	}
	names := []string{}
	for _, f := range e.Stack {
		names = append(names, f.Function)
	}
	joined := strings.Join(names, "->")
	if !strings.Contains(joined, "中层") || !strings.Contains(joined, "顶层") {
		t.Fatalf("调用栈应包含传播路径，实际：%s", joined)
	}
}

func TestWrapCause(t *testing.T) {
	cause := object.NewError("连接超时").WithCode("TIMEOUT")
	outer := object.NewError("请求失败").WithCode("REQUEST").WithCause(cause)
	if outer.Cause == nil || outer.Cause.Code != "TIMEOUT" {
		t.Fatal("原因链错误")
	}
	if !strings.Contains(outer.Trace(), "原因") {
		t.Fatalf("追踪信息应包含原因链：%s", outer.Trace())
	}
}
