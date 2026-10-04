package evaluator

import "testing"

func TestNoArgFunctionWithoutParens(t *testing.T) {
	src := `变量 n = 0
函数 计数()
    n++
结束
计数
n 等于 1`
	mustBool(t, src, true)
}

func TestNoArgMethodWithoutParens(t *testing.T) {
	src := `模型 A
    变量 x
结束
定义 A 的 方法 f()
    自己.x = 1
结束
变量 a = A()
a 的 f
a.x 等于 1`
	mustBool(t, src, true)
}

func TestNoArgMethodWithoutParensInArgument(t *testing.T) {
	src := `模型 A
    变量 x
结束
定义 A 的 方法 初始化(x 整数)
    自己.x = x
结束
定义 A 的 方法 f() -> 整数
    返回 自己.x
结束
变量 a = A(41)
函数 包装(n 整数) -> 整数
    返回 n + 1
结束
包装(a.f)`
	mustInt(t, src, 42)
}

func TestFieldWithoutParensStaysField(t *testing.T) {
	src := `模型 A
    变量 x
结束
变量 a = A()
a.x = 42
a 的 x`
	mustInt(t, src, 42)
}
