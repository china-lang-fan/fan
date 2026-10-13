package evaluator

import (
	"bytes"
	"strings"
	"testing"

	"fan/internal/object"
)

func mustInspect(t *testing.T, src string, want string) {
	t.Helper()
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("%q 运行错误：%v", src, err)
	}
	if res.Inspect() != want {
		t.Fatalf("%q 期望 %s，实际 %s", src, want, res.Inspect())
	}
}

func TestArrayLiteralEval(t *testing.T) {
	mustInspect(t, "[]", "[]")
	mustInspect(t, "[1, 2, 3]", "[1, 2, 3]")
	mustInspect(t, `[1, "二", 真]`, "[1, 二, 真]")
	mustInspect(t, "[[1, 2], [3]]", "[[1, 2], [3]]")
	mustInspect(t, "[1 加 1, 2 乘 3]", "[2, 6]")
}

func TestArrayKind(t *testing.T) {
	res, err := runSource(t, "[1]")
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Kind() != object.KindArray {
		t.Fatalf("类型应为数组，实际 %s", res.Kind())
	}
}

func TestArrayIndexRead(t *testing.T) {
	mustInt(t, "变量 数字 = [10, 20, 30]\n数字[0]", 10)
	mustInt(t, "变量 数字 = [10, 20, 30]\n数字[2]", 30)
	mustInt(t, "变量 嵌套 = [[1, 2], [3, 4]]\n嵌套[1][0]", 3)
	mustInt(t, "[10, 20][1]", 20)
}

func TestArrayIndexOutOfRange(t *testing.T) {
	mustError(t, "变量 数字 = [1, 2, 3]\n数字[5]", "下标越界")
	mustError(t, "变量 数字 = [1, 2, 3]\n数字[-1]", "下标越界")
	mustError(t, "[][0]", "下标越界")
}

func TestArrayIndexMustBeInt(t *testing.T) {
	mustError(t, "变量 数字 = [1, 2]\n数字[1.5]", "下标必须是整数")
	mustError(t, `变量 数字 = [1, 2]`+"\n"+`数字["零"]`, "下标必须是整数")
}

func TestIndexOnNonArray(t *testing.T) {
	mustError(t, "变量 甲 = 1\n甲[0]", "不支持下标访问")
}

func TestArrayIndexAssign(t *testing.T) {
	mustInt(t, "变量 数字 = [10, 20]\n数字[0] = 99\n数字[0]", 99)
	mustInt(t, "变量 数字 = [10, 20]\n数字[1] 为 88\n数字[1]", 88)
	mustInspect(t, "变量 数字 = [1, 2]\n数字[0] = 9\n数字", "[9, 2]")
}

func TestArrayIndexAssignOutOfRange(t *testing.T) {
	mustError(t, "变量 数字 = [1]\n数字[3] = 9", "下标越界")
}

func TestConstArrayElementMutable(t *testing.T) {
	mustInt(t, "常量 数字 = [1, 2, 3]\n数字[0] = 9\n数字[0]", 9)
}

func TestConstArrayRebindError(t *testing.T) {
	mustError(t, "常量 数字 = [1, 2]\n数字 = [3]", "常量")
}

func TestArrayReferenceSemantics(t *testing.T) {
	src := strings.Join([]string{
		`变量 甲 = [1, 2, 3]`,
		`变量 乙 = 甲`,
		`乙[0] = 99`,
		`甲[0]`,
	}, "\n")
	mustInt(t, src, 99)
}

func TestArrayConcat(t *testing.T) {
	mustInspect(t, "[1, 2] 加 [3]", "[1, 2, 3]")
	mustInspect(t, "[1] + [2] + [3]", "[1, 2, 3]")
	mustInspect(t, "[] 加 []", "[]")
}

func TestArrayConcatDoesNotMutate(t *testing.T) {
	src := strings.Join([]string{
		`变量 甲 = [1, 2]`,
		`变量 乙 = 甲 加 [3]`,
		`长度(甲)`,
	}, "\n")
	mustInt(t, src, 2)
}

func TestArrayDeepEqual(t *testing.T) {
	mustBool(t, "[1, 2] 等于 [1, 2]", true)
	mustBool(t, "[1, 2] 等于 [2, 1]", false)
	mustBool(t, "[1, 2] 等于 [1]", false)
	mustBool(t, "[] 等于 []", true)
	mustBool(t, "[[1]] 等于 [[1]]", true)
	mustBool(t, "[1, 2] 不等于 [1]", true)
}

func TestArrayCompareUnsupported(t *testing.T) {
	mustError(t, "[1] 小于 [2]", "不支持大小比较")
	mustError(t, "[1] 大于等于 [2]", "不支持大小比较")
}

func TestArrayArithmeticUnsupported(t *testing.T) {
	mustError(t, "[1] 减 [2]", "不支持")
	mustError(t, "[1] 乘 2", "不支持")
}

func TestArrayTypeCheck(t *testing.T) {
	mustInspect(t, `变量 数组 名单 = ["小明"]`+"\n名单", "[小明]")
	mustError(t, "变量 数组 名单 = 1", "类型不匹配")
	mustError(t, "变量 数组 名单 = [1]\n名单 = 2", "类型不匹配")
	mustError(t, "变量 整数 甲 = [1]", "类型不匹配")
}

func TestBuiltinLength(t *testing.T) {
	mustInt(t, "长度([1, 2, 3])", 3)
	mustInt(t, "长度([])", 0)
	mustInt(t, `长度("你好世界")`, 4)
	mustError(t, "长度(1)", "不支持")
	mustError(t, "长度()", "需要 1 个参数")
	mustError(t, "长度([1], [2])", "需要 1 个参数")
}

func TestBuiltinAppend(t *testing.T) {
	mustInspect(t, "追加([1, 2], 3)", "[1, 2, 3]")
	mustInspect(t, "追加([], 1, 2)", "[1, 2]")
	mustInspect(t, `追加([1], "二", 真)`, "[1, 二, 真]")
	mustError(t, "追加([1])", "至少需要 2 个参数")
	mustError(t, "追加(1, 2)", "必须是数组")
}

func TestBuiltinAppendDoesNotMutate(t *testing.T) {
	src := strings.Join([]string{
		`变量 甲 = [1]`,
		`追加(甲, 2)`,
		`长度(甲)`,
	}, "\n")
	mustInt(t, src, 1)
}

func TestBuiltinPrint(t *testing.T) {
	var buf bytes.Buffer
	if _, _, err := runSourceWithOutput(t, `打印([1, "二", 真])`, &buf); err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if got := buf.String(); got != "[1, 二, 真]\n" {
		t.Fatalf("打印输出不符：%q", got)
	}
}

func TestBuiltinPrintMultipleArgs(t *testing.T) {
	var buf bytes.Buffer
	if _, _, err := runSourceWithOutput(t, `打印(1, "甲", 真)`, &buf); err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if got := buf.String(); got != "1 甲 真\n" {
		t.Fatalf("打印输出不符：%q", got)
	}
}

func TestUnknownFunction(t *testing.T) {
	mustError(t, "未知函数名(1)", "未知函数")
}

func TestArrayInIfCondition(t *testing.T) {
	src := strings.Join([]string{
		`变量 名单 = [1, 2, 3]`,
		`变量 结果 = 0`,
		`如果 长度(名单) 大于 2 那么`,
		`    结果 = 1`,
		`结束`,
		`结果`,
	}, "\n")
	mustInt(t, src, 1)
}

func TestDictLiteral(t *testing.T) {
	mustInspect(t, `{}`, "{}")
	mustInspect(t, `{"a": 1, "b": 2}`, "{a: 1, b: 2}")
}

func TestDictGetSet(t *testing.T) {
	src := strings.Join([]string{
		`变量 d = {"a": 1}`,
		`d["b"] = 2`,
		`d["b"]`,
	}, "\n")
	mustInt(t, src, 2)
}

func TestDictMissingKey(t *testing.T) {
	mustInspect(t, `{}["x"]`, "空")
}

func TestDictLength(t *testing.T) {
	mustInt(t, `长度({"a": 1, "b": 2})`, 2)
	mustInt(t, `{"a": 1} 的 长度`, 1)
}

func TestDictKeysValues(t *testing.T) {
	mustInspect(t, `{"a": 1} 的 键`, "[a]")
	mustInspect(t, `{"a": 1} 的 值`, "[1]")
}

func TestDictForEach(t *testing.T) {
	src := strings.Join([]string{
		`变量 d = {"a": 1, "b": 2}`,
		`变量 和 = 0`,
		`遍历 d 中的 k v`,
		`    和 = 和 加 v`,
		`结束`,
		`和`,
	}, "\n")
	mustInt(t, src, 3)
}

func TestDictForEachValue(t *testing.T) {
	src := strings.Join([]string{
		`变量 d = {"a": 1, "b": 2}`,
		`变量 和 = 0`,
		`遍历 d 中的 v`,
		`    和 = 和 加 v`,
		`结束`,
		`和`,
	}, "\n")
	mustInt(t, src, 3)
}

func TestDictIntKeys(t *testing.T) {
	src := strings.Join([]string{
		`变量 d = {}`,
		`d[1] = "一"`,
		`d[2] = "二"`,
		`d[1]`,
	}, "\n")
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "一" {
		t.Fatalf("整数键读取失败：%s", res.Inspect())
	}
}

func TestDictEqual(t *testing.T) {
	mustBool(t, `{"a": 1} 等于 {"a": 1}`, true)
	mustBool(t, `{"a": 1} 等于 {"a": 2}`, false)
}

func TestDictBadKey(t *testing.T) {
	mustError(t, `{[1]: 2}`, "字典键")
}
