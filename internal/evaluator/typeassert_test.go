package evaluator

import (
	"testing"

	"fan/internal/object"
)

func isTruthy(o object.Object) bool {
	if b, ok := o.(*object.Bool); ok {
		return b.Value
	}
	return false
}

const isCastModelsSrc = `
定义 模型 动物
    变量 字符串 名
结束
定义 模型 狗
    变量 字符串 名
结束
定义 狗 的 方法 初始化(名 字符串)
    自己.名 = 名
结束
定义 狗 的 方法 说话() -> 字符串
    返回 "汪"
结束
定义 模型 牧羊犬
    变量 字符串 名
    变量 整数 级别
结束
定义 牧羊犬 的 方法 初始化(名 字符串, 级别 整数)
    自己.名 = 名
    自己.级别 = 级别
结束
定义 牧羊犬 的 方法 说话() -> 字符串
    返回 "汪汪"
结束
接口 说话者
    方法 说话() -> 字符串
结束
`

func evalBool(t *testing.T, src string) bool {
	t.Helper()
	res, err := runSource(t, isCastModelsSrc+src)
	if err != nil {
		t.Fatal(err)
	}
	return isTruthy(res)
}

func TestIsKeyword(t *testing.T) {
	cases := map[string]bool{
		`变量 a = 狗("x")
a 是 狗`: true,
		`变量 a = 狗("x")
a 是 说话者`: true,
		`变量 a = 动物()
a 是 狗`: false,
		`变量 a = 动物()
a 是 说话者`: false,
		`变量 a = 牧羊犬("x", 1)
a 是 狗`: false,
		`"x" 是 字符串`: true,
		`1 是 整数`:    true,
		`'中' 是 字符`:  true,
		`1.5 是 整数`:  false,
	}
	for src, want := range cases {
		if got := evalBool(t, src); got != want {
			t.Fatalf("%q 期望 %v，实际 %v", src, want, got)
		}
	}
}

func TestIsKeywordPrecedence(t *testing.T) {
	// instanceof 与比较同级：1 是 整数 且 2 是 整数 应能正常求值
	res, err := runSource(t, isCastModelsSrc+`
变量 a = 狗("x")
a 是 狗 且 a 是 说话者
`)
	if err != nil {
		t.Fatal(err)
	}
	if !isTruthy(res) {
		t.Fatalf("是 与 且 的优先级组合错误：%v", res.Inspect())
	}
}

func TestIsKeywordInIf(t *testing.T) {
	res, err := runSource(t, isCastModelsSrc+`
变量 a = 狗("x")
如果 a 是 狗 那么
    返回 "是狗"
结束
返回 "不是"
`)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inspect() != "是狗" {
		t.Fatalf("条件中 是 判断错误：%s", res.Inspect())
	}
}

func TestCastSuccessModel(t *testing.T) {
}

func TestCastSuccess(t *testing.T) {
	res, err := runSource(t, isCastModelsSrc+`
变量 a = 狗("阿黄")
变量 u、ok = a 转为 说话者
如果 ok 那么
    返回 u.说话()
结束
返回 "失败"
`)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := res.(*object.String); !ok || s.Value != "汪" {
		t.Fatalf("转为 接口成功后调用错误：%s", res.Inspect())
	}
}

func TestCastFailure(t *testing.T) {
	res, err := runSource(t, isCastModelsSrc+`
变量 a = 动物()
变量 u、ok = a 转为 说话者
返回 ok
`)
	if err != nil {
		t.Fatal(err)
	}
	if isTruthy(res) {
		t.Fatalf("不满足接口应返回 假")
	}
}

func TestCastPrimitive(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`变量 n、ok = "42" 转为 整数
返回 n`, "42"},
		{`变量 s、ok = 65 转为 字符串
返回 s`, "65"},
		{`变量 f、ok = 3 转为 小数
返回 f`, "3"},
		{`变量 n、ok = "abc" 转为 整数
返回 ok`, "假"},
	}
	for _, c := range cases {
		res, err := runSource(t, c.src)
		if err != nil {
			t.Fatalf("%s: %v", c.src, err)
		}
		if res.Inspect() != c.want {
			t.Fatalf("%s 期望 %s，实际 %s", c.src, c.want, res.Inspect())
		}
	}
}

func TestCastSameType(t *testing.T) {
	res, err := runSource(t, isCastModelsSrc+`
变量 a = 狗("x")
变量 u、ok = a 转为 狗
返回 ok
`)
	if err != nil {
		t.Fatal(err)
	}
	if !isTruthy(res) {
		t.Fatalf("同类型断言应成功")
	}
}
