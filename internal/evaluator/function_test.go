package evaluator

import (
	"strings"
	"testing"
)

func TestFunctionAdd(t *testing.T) {
	src := strings.Join([]string{
		`函数 求和（甲，乙）`,
		`    返回 甲 加 乙`,
		`结束`,
		`求和(2, 3)`,
	}, "\n")
	mustInt(t, src, 5)
}

func TestFunctionImplicitCall(t *testing.T) {
	src := strings.Join([]string{
		`函数 问候（名字）`,
		`    打印("你好")`,
		`结束`,
		`问候 "小明"`,
	}, "\n")
	if _, err := runSource(t, src); err != nil {
		t.Fatalf("运行错误：%v", err)
	}
}

func TestAnonymousFunction(t *testing.T) {
	src := strings.Join([]string{
		`变量 平方 = 函数（数）`,
		`    返回 数 乘 数`,
		`结束`,
		`平方(4)`,
	}, "\n")
	mustInt(t, src, 16)
}

func TestClosure(t *testing.T) {
	src := strings.Join([]string{
		`函数 计数器（）`,
		`    变量 值 = 0`,
		`    返回 函数（）`,
		`        值 = 值 加 1`,
		`        返回 值`,
		`    结束`,
		`结束`,
		`变量 下一个 = 计数器()`,
		`下一个()`,
		`下一个()`,
	}, "\n")
	mustInt(t, src, 2)
}

func TestRecursion(t *testing.T) {
	src := strings.Join([]string{
		`函数 阶乘(n)`,
		`    如果 n 小于等于 1 那么`,
		`        返回 1`,
		`    结束`,
		`    返回 n 乘 阶乘(n 减 1)`,
		`结束`,
		`阶乘(5)`,
	}, "\n")
	mustInt(t, src, 120)
}

func TestReturnWithoutValue(t *testing.T) {
	src := strings.Join([]string{
		`函数 空跑()`,
		`    返回`,
		`结束`,
		`空跑()`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Kind() != "空" {
		t.Fatalf("无值返回应为 空，实际 %s", res.Inspect())
	}
}

func TestImplicitReturnAtEnd(t *testing.T) {
	src := strings.Join([]string{
		`函数 常数值()`,
		`    42`,
		`结束`,
		`常数值()`,
	}, "\n")
	mustInt(t, src, 42)
}

func TestFunctionWrongArgCount(t *testing.T) {
	mustError(t, "函数 求和(甲,乙)\n    返回 甲 加 乙\n结束\n求和(1)", "参数数量不符")
}

func TestCallNonFunction(t *testing.T) {
	mustError(t, "变量 甲 = 1\n甲(2)", "不是函数")
}

func TestCallUnknownFunction(t *testing.T) {
	mustError(t, "未知函数(1)", "未知函数")
}

func TestReturnOnlyInFunction(t *testing.T) {
	src := strings.Join([]string{
		`变量 结果 = 0`,
		`函数 测试()`,
		`    变量 i = 0`,
		`    当 i 小于 5 循环`,
		`        如果 i 等于 3 那么`,
		`            返回 i`,
		`        结束`,
		`        i = i 加 1`,
		`    结束`,
		`结束`,
		`测试()`,
	}, "\n")
	mustInt(t, src, 3)
}

func TestFunctionAsArgument(t *testing.T) {
	src := strings.Join([]string{
		`函数 应用(f, x)`,
		`    返回 f(x)`,
		`结束`,
		`函数 双倍(n)`,
		`    返回 n 乘 2`,
		`结束`,
		`应用(双倍, 5)`,
	}, "\n")
	mustInt(t, src, 10)
}

func TestMemberLength(t *testing.T) {
	src := strings.Join([]string{
		`变量 名单 = [1, 2, 3]`,
		`名单 的 长度`,
	}, "\n")
	mustInt(t, src, 3)
}

func TestMemberMethodCall(t *testing.T) {
	src := strings.Join([]string{
		`变量 名单 = [1, 2, 3]`,
		`名单 的 反转()`,
	}, "\n")
	mustInspect(t, src, "[3, 2, 1]")
}

func TestImplicitReceiverCall(t *testing.T) {
	src := strings.Join([]string{
		`变量 名单 = [1, 2, 3]`,
		`变量 结果 = 名单 的 反转()`,
		`结果 的 长度`,
	}, "\n")
	mustInt(t, src, 3)
}

func TestFunctionWithArray(t *testing.T) {
	src := strings.Join([]string{
		`函数 求和数组(列)`,
		`    变量 合计 = 0`,
		`    变量 i = 0`,
		`    当 i 小于 长度(列) 循环`,
		`        合计 = 合计 加 列[i]`,
		`        i = i 加 1`,
		`    结束`,
		`    返回 合计`,
		`结束`,
		`求和数组([1, 2, 3, 4, 5])`,
	}, "\n")
	mustInt(t, src, 15)
}

func TestTypedParameterCheck(t *testing.T) {
	src := "函数 问候(名字 字符串)\n    打印 名字\n结束\n问候(123)"
	mustError(t, src, "参数 名字 类型不匹配")
}

func TestReturnTypeCheck(t *testing.T) {
	src := "函数 数值() -> 整数\n    返回 \"abc\"\n结束\n数值()"
	mustError(t, src, "返回值类型不匹配")
}

func TestTypedFunctionSuccess(t *testing.T) {
	src := "函数 求和(甲 整数、乙 整数) -> 整数\n    返回 甲 加 乙\n结束\n求和(2, 3)"
	mustInt(t, src, 5)
}

func TestDefineFunction(t *testing.T) {
	src := "定义 函数 双倍(数 整数) -> 整数\n    返回 数 乘 2\n结束\n双倍(4)"
	mustInt(t, src, 8)
}

func TestMultiReturnAndDecl(t *testing.T) {
	src := strings.Join([]string{
		`函数 商余(a 整数，b 整数) -> (整数，整数)`,
		`    返回 a / b、a % b`,
		`结束`,
		`变量 商、余数 = 商余(10, 3)`,
		`商`,
	}, "\n")
	mustInt(t, src, 3)
}

func TestMultiReturnSecondValue(t *testing.T) {
	src := strings.Join([]string{
		`函数 商余(a 整数，b 整数) -> (整数，整数)`,
		`    返回 a / b、a % b`,
		`结束`,
		`变量 商、余数 = 商余(10, 3)`,
		`余数`,
	}, "\n")
	mustInt(t, src, 1)
}

func TestMultiReturnSpread(t *testing.T) {
	src := strings.Join([]string{
		`函数 坐标() -> (整数，整数)`,
		`    返回 1、2`,
		`结束`,
		`函数 相加(a 整数，b 整数) -> 整数`,
		`    返回 a 加 b`,
		`结束`,
		`相加(坐标())`,
	}, "\n")
	mustInt(t, src, 3)
}

func TestMultiAssignExisting(t *testing.T) {
	src := strings.Join([]string{
		`函数 取值() -> (整数，字符串)`,
		`    返回 1、"x"`,
		`结束`,
		`变量 a = 0`,
		`变量 b = ""`,
		`a、b = 取值()`,
		`a`,
	}, "\n")
	mustInt(t, src, 1)
}

func TestMultiReturnCountMismatch(t *testing.T) {
	src := "函数 f() -> (整数，整数)\n    返回 1\n结束\nf()"
	mustError(t, src, "返回值数量不符")
}

func TestMultiAssignCountMismatch(t *testing.T) {
	src := "函数 f() -> 整数\n    返回 1\n结束\n变量 a、b = f()"
	mustError(t, src, "多目标赋值")
}

func TestMultiReturnNoType(t *testing.T) {
	src := "函数 f()\n    返回 1、\"x\"、真\n结束\n变量 a、b、c = f()\nb"
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "x" {
		t.Fatalf("多返回值无类型声明应工作，实际 %s", res.Inspect())
	}
}

func TestSingleReturnNotWrapped(t *testing.T) {
	src := "函数 f() -> 整数\n    返回 1\n结束\n变量 x = f()\nx"
	mustInt(t, src, 1)
}

func TestErrorConstructor(t *testing.T) {
	src := `错误("boom").消息`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "boom" {
		t.Fatalf("错误消息应为 boom，实际 %s", res.Inspect())
	}
}

func TestErrorReturnAndCheck(t *testing.T) {
	src := strings.Join([]string{
		`函数 除法(a 整数，b 整数) -> (整数，错误)`,
		`    如果 b 等于 0 那么`,
		`        返回 0、错误("零")`,
		`    结束`,
		`    返回 a / b、空`,
		`结束`,
		`函数 安全除法() -> (整数，错误)`,
		`    返回 检查 除法(10, 0)`,
		`结束`,
		`变量 r、e = 安全除法()`,
		`e.消息`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "零" {
		t.Fatalf("应返回错误，实际 %s", res.Inspect())
	}
}

func TestCheckPropagation(t *testing.T) {
	src := strings.Join([]string{
		`函数 内层() -> (整数，错误)`,
		`    返回 0、错误("坏")`,
		`结束`,
		`函数 外层() -> (整数，错误)`,
		`    变量 x = 检查 内层()`,
		`    返回 x、空`,
		`结束`,
		`变量 r、e = 外层()`,
		`e.消息`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "坏" {
		t.Fatalf("错误应传播，实际 %s", res.Inspect())
	}
}

func TestCheckSuccess(t *testing.T) {
	src := strings.Join([]string{
		`函数 f() -> (整数，错误)`,
		`    返回 42、空`,
		`结束`,
		`变量 x = 检查 f()`,
		`x`,
	}, "\n")
	mustInt(t, src, 42)
}

func TestErrorEqualsNil(t *testing.T) {
	src := strings.Join([]string{
		`函数 f() -> 错误`,
		`    返回 空`,
		`结束`,
		`f() 等于 空`,
	}, "\n")
	mustBool(t, src, true)
}

func TestCompoundAssignment(t *testing.T) {
	mustInt(t, "变量 x = 10\nx += 5\nx", 15)
	mustInt(t, "变量 x = 10\nx -= 3\nx", 7)
	mustInt(t, "变量 x = 4\nx *= 3\nx", 12)
	mustInt(t, "变量 x = 20\nx /= 4\nx", 5)
	mustInt(t, "变量 数 = [1, 2, 3]\n数[1] 加等于 4\n数[1]", 6)
}

func TestCompoundAssignmentArrayAndString(t *testing.T) {
	mustInspect(t, "变量 x = [1]\nx += [2, 3]\nx", "[1, 2, 3]")
	res := strings.Join([]string{
		`变量 x = "你"`,
		`x 加等于 "好"`,
		`x`,
	}, "\n")
	mustInspect(t, res, "你好")
}

func TestIncrementDecrement(t *testing.T) {
	mustInt(t, "变量 i = 1\n变量 旧 = i++\n旧", 1)
	mustInt(t, "变量 i = 1\ni++\ni", 2)
	mustInt(t, "变量 i = 1\n++i", 2)
	mustInt(t, "变量 i = 2\n变量 旧 = i--\n旧", 2)
	mustInt(t, "变量 i = 2\ni--\ni", 1)
	mustInt(t, "变量 i = 2\n--i", 1)
}

func TestVariadicParameters(t *testing.T) {
	src := strings.Join([]string{
		`函数 求和(开头 整数，其余 ...整数)`,
		`    变量 合计 = 开头`,
		`    遍历 其余 中的 数`,
		`        合计 加等于 数`,
		`    结束`,
		`    返回 合计`,
		`结束`,
		`求和(1, 2, 3, 4)`,
	}, "\n")
	mustInt(t, src, 10)
}

func TestVariadicNoExtra(t *testing.T) {
	src := strings.Join([]string{
		`函数 f(x 整数，其余 ...整数)`,
		`    返回 长度(其余)`,
		`结束`,
		`f(1)`,
	}, "\n")
	mustInt(t, src, 0)
}

func TestVariadicTypeCheck(t *testing.T) {
	src := "函数 f(其余 ...整数)\n结束\nf(1, 真)"
	mustError(t, src, "参数 其余 类型不匹配")
}

func TestDefaultParameters(t *testing.T) {
	src := strings.Join([]string{
		`函数 问候(名字 字符串, 称呼 字符串 = "你好")`,
		`    返回 称呼 加 "，" 加 名字`,
		`结束`,
		`问候("小明")`,
	}, "\n")
	mustInspect(t, src, "你好，小明")
	mustInspect(t, "函数 f(a 整数 = 2, b 整数 = 3)\n返回 a 加 b\n结束\nf(10)", "13")
	mustInspect(t, "函数 f(a 整数 = 2, b 整数 = 3)\n返回 a 加 b\n结束\nf()", "5")
}

func TestDefaultParameterEvaluatedEachCall(t *testing.T) {
	src := strings.Join([]string{
		`变量 基数 = 1`,
		`函数 f(x = 基数)`,
		`    返回 x`,
		`结束`,
		`基数 = 9`,
		`f()`,
	}, "\n")
	mustInt(t, src, 9)
}
