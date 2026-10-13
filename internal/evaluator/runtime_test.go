package evaluator

import (
	"bytes"
	"strings"
	"testing"

	"fan/internal/parser"
)

func runInRuntime(t *testing.T, rt *Runtime, src string) {
	t.Helper()
	prog, errs := parser.ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	env := NewEnvironmentForRuntime(rt)
	if _, err := Eval(prog, env); err != nil {
		t.Fatalf("运行错误：%v", err)
	}
}

// TestRuntimeStdoutIsolation 验证两个运行实例的打印输出互不干扰。
func TestRuntimeStdoutIsolation(t *testing.T) {
	var a, b bytes.Buffer
	rtA := NewRuntime()
	rtA.SetStdout(&a)
	rtB := NewRuntime()
	rtB.SetStdout(&b)

	runInRuntime(t, rtA, `打印("A")`)
	runInRuntime(t, rtB, `打印("B")`)
	runInRuntime(t, rtA, `打印("A2")`)

	if strings.TrimSpace(a.String()) != "A\nA2" {
		t.Fatalf("实例 A 输出错误：%q", a.String())
	}
	if strings.TrimSpace(b.String()) != "B" {
		t.Fatalf("实例 B 输出错误：%q", b.String())
	}
}

// TestRuntimeTagHandlerIsolation 验证“注册标签处理”只影响当前实例。
func TestRuntimeTagHandlerIsolation(t *testing.T) {
	modelAndUse := `模型 自定义
    变量 值
结束
@自定义
定义 函数 测试()
    返回 空
结束
测试()`
	register := `函数 处理(上下文)
    打印("实例A处理器")
结束
注册标签处理("自定义", 处理)`

	rtA := NewRuntime()
	var bufA bytes.Buffer
	rtA.SetStdout(&bufA)
	runInRuntime(t, rtA, register+"\n"+modelAndUse)
	if !strings.Contains(bufA.String(), "实例A处理器") {
		t.Fatalf("实例 A 应执行自定义标签处理器，输出：%q", bufA.String())
	}

	rtB := NewRuntime()
	var bufB bytes.Buffer
	rtB.SetStdout(&bufB)
	runInRuntime(t, rtB, modelAndUse)
	if strings.Contains(bufB.String(), "实例A处理器") {
		t.Fatalf("实例 B 不应执行实例 A 注册的标签处理器，输出：%q", bufB.String())
	}
}
