package evaluator

import (
	"testing"

	"fan/internal/object"
)

func TestSliceArray(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`[1, 2, 3, 4, 5][1:3]`, "[2, 3]"},
		{`[1, 2, 3, 4, 5][2:]`, "[3, 4, 5]"},
		{`[1, 2, 3, 4, 5][:3]`, "[1, 2, 3]"},
		{`[1, 2, 3, 4, 5][:]`, "[1, 2, 3, 4, 5]"},
		{`[1, 2, 3, 4, 5][0:0]`, "[]"},
	}
	for _, c := range cases {
		res, err := runSource(t, c.src)
		if err != nil {
			t.Fatalf("%s 错误：%v", c.src, err)
		}
		if got := res.Inspect(); got != c.want {
			t.Errorf("%s 期望 %s，得到 %s", c.src, c.want, got)
		}
	}
}

func TestSliceStringByRune(t *testing.T) {
	res, err := runSource(t, `"你好凡语言"[2:4]`)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inspect() != "凡语" {
		t.Fatalf("字符串应按字符切片，得到 %s", res.Inspect())
	}
}

func TestSliceOutOfBounds(t *testing.T) {
	if _, err := runSource(t, `[1, 2, 3][2:1]`); err == nil {
		t.Fatal("low > high 应报越界")
	}
	if _, err := runSource(t, `[1, 2, 3][:5]`); err == nil {
		t.Fatal("超出长度应报越界")
	}
}

func TestSliceDictFails(t *testing.T) {
	if _, err := runSource(t, `{"a": 1}[:1]`); err == nil {
		t.Fatal("字典不支持切片")
	}
}

func TestArraySpreadLiteral(t *testing.T) {
	res, err := runSource(t, `
变量 前 = [1, 2]
变量 后 = [4, 5]
[0, ...前, 3, ...后]`)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inspect() != "[0, 1, 2, 3, 4, 5]" {
		t.Fatalf("数组展开错误：%s", res.Inspect())
	}
}

func TestArraySpreadNonArrayFails(t *testing.T) {
	if _, err := runSource(t, `[...1]`); err == nil {
		t.Fatal("非数组不应能展开")
	}
}

func TestSpreadCallArgs(t *testing.T) {
	res, err := runSource(t, `
函数 求和(数 ...整数) -> 整数
    变量 总 = 0
    遍历 数 中的 n
        总 = 总 加 n
    结束
    返回 总
结束
求和(1, ...[2, 3], ...[4])`)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inspect() != "10" {
		t.Fatalf("调用展开错误：%s", res.Inspect())
	}
}

func TestSpreadNonVariadicCall(t *testing.T) {
	res, err := runSource(t, `
函数 三(a, b, c)
    返回 [a, b, c]
结束
三(...[4, 5, 6])`)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inspect() != "[4, 5, 6]" {
		t.Fatalf("普通函数展开错误：%s", res.Inspect())
	}
}

func TestDictSpread(t *testing.T) {
	res, err := runSource(t, `
变量 a = {"x": 1, "y": 2}
变量 b = {"y": 9, "z": 3}
{...a, ...b, "w": 4}`)
	if err != nil {
		t.Fatal(err)
	}
	dict, ok := res.(*object.Dict)
	if !ok {
		t.Fatalf("期望字典，得到 %T", res)
	}
	checks := map[string]int64{"x": 1, "y": 9, "z": 3, "w": 4}
	for k, want := range checks {
		key := &object.String{Value: k}
		v, exists := dict.Get(key)
		if !exists {
			t.Fatalf("缺少键 %s", k)
		}
		if v.Inspect() != objectIntText(want) {
			t.Errorf("键 %s 期望 %d，得到 %s", k, want, v.Inspect())
		}
	}
}

func objectIntText(n int64) string {
	return (&object.Integer{Value: n}).Inspect()
}

func TestDictSpreadNonDictFails(t *testing.T) {
	if _, err := runSource(t, `{...[1, 2]}`); err == nil {
		t.Fatal("非字典不应能在字典字面量展开")
	}
}

func TestSliceDoesNotMutateSource(t *testing.T) {
	res, err := runSource(t, `
变量 原 = [1, 2, 3]
变量 段 = 原[1:2]
段[0] = 99
原`)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inspect() != "[1, 2, 3]" {
		t.Fatalf("切片应为浅拷贝的新数组，原数组被改：%s", res.Inspect())
	}
}
