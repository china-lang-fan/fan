package evaluator

import (
	"strings"
	"testing"

	"fan/internal/object"
)

func TestDeferLIFO(t *testing.T) {
	src := strings.Join([]string{
		`函数 f()`,
		`    稍后 追加记录("第二")`,
		`    稍后 追加记录("第一")`,
		`结束`,
		`变量 记录 = []`,
		`函数 追加记录(文本 字符串)`,
		`    记录 = 追加(记录, 文本)`,
		`结束`,
		`f()`,
		`记录`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != `[第一, 第二]` {
		t.Fatalf("稍后应逆序执行，实际 %s", res.Inspect())
	}
}

func TestThrowRecover(t *testing.T) {
	src := strings.Join([]string{
		`函数 f()`,
		`    稍后 函数()`,
		`        全局结果 = 捕获()`,
		`    结束`,
		`    抛出 "失败"`,
		`结束`,
		`变量 全局结果 = 空`,
		`f()`,
		`全局结果`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != `失败` {
		t.Fatalf("应捕获抛出值，实际 %s", res.Inspect())
	}
}

func TestThrowAnyInstance(t *testing.T) {
	src := strings.Join([]string{
		`模型 异常`,
		`    变量 字符串 消息`,
		`结束`,
		`函数 f()`,
		`    稍后 函数()`,
		`        变量 值 = 捕获()`,
		`        全局消息 = 值.消息`,
		`    结束`,
		`    变量 e = 异常()`,
		`    e.消息 = "自定义"`,
		`    抛出 e`,
		`结束`,
		`变量 全局消息 = 空`,
		`f()`,
		`全局消息`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != `自定义` {
		t.Fatalf("应捕获模型实例，实际 %s", res.Inspect())
	}
}

func TestDeferImmediateArguments(t *testing.T) {
	src := strings.Join([]string{
		`变量 结果 = ""`,
		`函数 记录(文本 字符串)`,
		`    结果 = 文本`,
		`结束`,
		`函数 f()`,
		`    变量 文本 = "旧"`,
		`    稍后 记录(文本)`,
		`    文本 = "新"`,
		`结束`,
		`f()`,
		`结果`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != `旧` {
		t.Fatalf("稍后参数应立即求值，实际 %s", res.Inspect())
	}
}

func TestNestedThrowRecover(t *testing.T) {
	src := strings.Join([]string{
		`函数 内层()`,
		`    抛出 123`,
		`结束`,
		`函数 外层()`,
		`    稍后 函数()`,
		`        全局结果 = 捕获()`,
		`    结束`,
		`    内层()`,
		`结束`,
		`变量 全局结果 = 空`,
		`外层()`,
		`全局结果`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != `123` {
		t.Fatalf("应捕获嵌套抛出值，实际 %s", res.Inspect())
	}
}

func TestRecoverOnlyInDefer(t *testing.T) {
	src := strings.Join([]string{
		`函数 f()`,
		`    返回 捕获()`,
		`结束`,
		`f()`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Kind() != object.KindNil {
		t.Fatalf("非稍后中的捕获应为空，实际 %s", res.Inspect())
	}
}

func TestUncaughtThrow(t *testing.T) {
	src := `抛出 "致命"`
	if _, err := runSource(t, src); err == nil {
		t.Fatal("未捕获的抛出应返回错误")
	}
}
