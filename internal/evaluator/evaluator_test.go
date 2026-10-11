package evaluator

import (
	"strings"
	"testing"

	"fan/internal/object"
	"fan/internal/parser"
)

func runSource(t *testing.T, src string) (object.Object, error) {
	t.Helper()
	prog, errs := parser.ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	return Eval(prog, NewEnvironment())
}

func mustInt(t *testing.T, src string, want int64) {
	t.Helper()
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("%q 运行错误：%v", src, err)
	}
	i, ok := res.(*object.Integer)
	if !ok {
		t.Fatalf("%q 期望整数，实际 %T：%s", src, res, res.Inspect())
	}
	if i.Value != want {
		t.Fatalf("%q 期望 %d，实际 %d", src, want, i.Value)
	}
}

func mustFloat(t *testing.T, src string, want float64) {
	t.Helper()
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("%q 运行错误：%v", src, err)
	}
	f, ok := res.(*object.Float)
	if !ok {
		t.Fatalf("%q 期望小数，实际 %T：%s", src, res, res.Inspect())
	}
	if f.Value != want {
		t.Fatalf("%q 期望 %g，实际 %g", src, want, f.Value)
	}
}

func mustString(t *testing.T, src string, want string) {
	t.Helper()
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("%q 运行错误：%v", src, err)
	}
	s, ok := res.(*object.String)
	if !ok {
		t.Fatalf("%q 期望字符串，实际 %T：%s", src, res, res.Inspect())
	}
	if s.Value != want {
		t.Fatalf("%q 期望 %q，实际 %q", src, want, s.Value)
	}
}

func mustBool(t *testing.T, src string, want bool) {
	t.Helper()
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("%q 运行错误：%v", src, err)
	}
	b, ok := res.(*object.Bool)
	if !ok {
		t.Fatalf("%q 期望布尔，实际 %T：%s", src, res, res.Inspect())
	}
	if b.Value != want {
		t.Fatalf("%q 期望 %v，实际 %v", src, want, b.Value)
	}
}

func mustError(t *testing.T, src string, contains string) {
	t.Helper()
	_, err := runSource(t, src)
	if err == nil {
		t.Fatalf("%q 期望报错，实际成功", src)
	}
	if !strings.Contains(err.Error(), contains) {
		t.Fatalf("%q 报错应包含 %q，实际 %q", src, contains, err.Error())
	}
}

func TestArithmeticInteger(t *testing.T) {
	mustInt(t, "1 + 2", 3)
	mustInt(t, "1 加 2", 3)
	mustInt(t, "10 - 3", 7)
	mustInt(t, "10 减 3", 7)
	mustInt(t, "3 * 4", 12)
	mustInt(t, "3 乘 4", 12)
	mustInt(t, "10 / 3", 3)
	mustInt(t, "10 除 3", 3)
	mustInt(t, "10 % 3", 1)
	mustInt(t, "10 取余 3", 1)
	mustInt(t, "-5", -5)
	mustInt(t, "1 + 2 * 3", 7)
	mustInt(t, "(1 + 2) * 3", 9)
}

func TestArithmeticFloatPromotion(t *testing.T) {
	mustFloat(t, "1 + 2.0", 3.0)
	mustFloat(t, "1.0 + 2", 3.0)
	mustFloat(t, "5.0 / 2", 2.5)
	mustFloat(t, "5 / 2.0", 2.5)
}

func TestStringConcat(t *testing.T) {
	mustString(t, `"你" + "好"`, "你好")
	mustString(t, `"你" 加 "好"`, "你好")
	mustString(t, `变量 a = "fan" 变量 b = "lang" a 加 b`, "fanlang")
	mustString(t, `"" + "x"`, "x")
}

func TestStringConcatRequiresString(t *testing.T) {
	mustError(t, `"a" + 1`, "加法不支持")
	mustError(t, `1 + "a"`, "加法不支持")
}

func TestDivisionByZero(t *testing.T) {
	mustError(t, "1 / 0", "除数不能为零")
	mustError(t, "1.0 / 0", "除数不能为零")
	mustError(t, "1 % 0", "除数不能为零")
}

func TestModRequiresInt(t *testing.T) {
	mustError(t, "1.5 % 2", "取余仅支持整数")
	mustError(t, "5 % 2.0", "取余仅支持整数")
}

func TestComparison(t *testing.T) {
	mustBool(t, "1 == 1", true)
	mustBool(t, "1 等于 1", true)
	mustBool(t, "1 != 2", true)
	mustBool(t, "1 不等于 2", true)
	mustBool(t, "1 < 2", true)
	mustBool(t, "1 小于 2", true)
	mustBool(t, "2 > 1", true)
	mustBool(t, "1 <= 1", true)
	mustBool(t, "1 >= 1", true)
	mustBool(t, "1 大于等于 1", true)
	mustBool(t, `"a" == "a"`, true)
	mustBool(t, `"a" != "b"`, true)
	mustBool(t, "真 == 真", true)
}

func TestComparisonTypeMismatch(t *testing.T) {
	mustError(t, `1 == "1"`, "不同类型不可比较")
	mustError(t, `真 == 1`, "不同类型不可比较")
}

func TestTernary(t *testing.T) {
	mustInt(t, "真 ? 1 : 2", 1)
	mustInt(t, "假 ? 1 : 2", 2)
	mustInt(t, "1 大于 2 ? 10 : 20", 20)
	mustInt(t, "真 ? 1 + 2 : 3", 3)
	mustInt(t, "变量 x = 真\nx ？ 1 ： 2", 1)
}

func TestTernaryLazy(t *testing.T) {
	mustInt(t, "真 ? 1 : 1 除 0", 1)
	mustInt(t, "假 ? 1 除 0 : 2", 2)
}

func TestTernaryConditionType(t *testing.T) {
	mustError(t, "1 ? 2 : 3", "布尔")
}

func TestLogic(t *testing.T) {
	mustBool(t, "真 且 真", true)
	mustBool(t, "真 && 假", false)
	mustBool(t, "假 || 真", true)
	mustBool(t, "非 假", true)
	mustBool(t, "!真", false)
	mustBool(t, "真 且 假 或 真", true)
}

func TestLogicRequiresBool(t *testing.T) {
	mustError(t, "1 且 真", "布尔")
	mustError(t, "真 且 1", "布尔")
	mustError(t, "非 1", "布尔")
}

func TestVarDeclAndUse(t *testing.T) {
	mustInt(t, "定义 变量 整数 年龄 为 18\n年龄 + 2", 20)
	mustInt(t, "变量 甲 = 3\n变量 乙 = 4\n甲 * 乙", 12)
	mustFloat(t, "定义 常量 小数 圆周率 为 3.14\n圆周率", 3.14)
	mustInt(t, "名字数 = 10\n名字数 + 5", 15)
}

func TestVarReassignment(t *testing.T) {
	mustInt(t, "变量 年龄 = 18\n年龄 = 20\n年龄", 20)
	mustInt(t, "变量 年龄 = 18\n年龄 为 20\n年龄", 20)
}

func TestConstReassignmentError(t *testing.T) {
	mustError(t, "常量 圆周率 = 3.14\n圆周率 = 3.15", "常量")
}

func TestTypeMismatchOnAssign(t *testing.T) {
	mustError(t, `变量 整数 年龄 = 18`+"\n"+`年龄 = "二十"`, "类型不匹配")
	mustError(t, `变量 整数 年龄 = "十八"`, "类型不匹配")
}

func TestDuplicateDeclaration(t *testing.T) {
	mustError(t, "变量 甲 = 1\n变量 甲 = 2", "已声明")
}

func TestUndeclared(t *testing.T) {
	mustError(t, "未定义 + 1", "未声明")
}

func TestReadablePrefixKeywords(t *testing.T) {
	mustInt(t, "从 长度([1, 2, 3])", 3)
	mustInt(t, "变量 数据 = [1, 2]\n通过 长度(数据)", 2)
	mustInt(t, "变量 数据 = [7, 8]\n从 数据[0]", 7)
}

func TestShortCircuit(t *testing.T) {
	mustBool(t, "假 且 (1 除 0 大于 0)", false)
	mustBool(t, "真 或 (1 除 0 大于 0)", true)
}
