package evaluator

import (
	"testing"

	"fan/internal/object"
)

const namedTypeSrc = `
类型 状态 整数

常量 状态 无效 = 状态(0)
常量 状态 有效 = 状态(1)
`

func TestNamedTypeConversionCall(t *testing.T) {
	res, err := runSource(t, namedTypeSrc+`状态(2)`)
	if err != nil {
		t.Fatal(err)
	}
	n, ok := res.(*object.Named)
	if !ok {
		t.Fatalf("期望自定义类型值，实际 %T", res)
	}
	if n.TypeName != "状态" {
		t.Fatalf("类型名错误：%s", n.TypeName)
	}
	if i, ok := n.Value.(*object.Integer); !ok || i.Value != 2 {
		t.Fatalf("底层值错误：%v", n.Value)
	}
}

func TestNamedTypeStrictIdentity(t *testing.T) {
	src := `类型 状态 整数
变量 状态 x = 1
x`
	mustError(t, src, "需显式转换")
}

func TestNamedTypeExplicitDecl(t *testing.T) {
	res, err := runSource(t, namedTypeSrc+`
变量 状态 x = 状态(1)
x`)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := res.(*object.Named); !ok || n.TypeName != "状态" {
		t.Fatalf("声明类型错误：%v", res)
	}
}

func TestNamedTypeIsKeyword(t *testing.T) {
	cases := map[string]bool{
		`有效 是 状态`:  true,
		`有效 是 整数`:  true,
		`1 是 状态`:   false,
		`无效 是 字符串`: false,
	}
	for src, want := range cases {
		res, err := runSource(t, namedTypeSrc+src)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		got := isTruthy(res)
		if got != want {
			t.Fatalf("%s 期望 %v，实际 %v", src, want, got)
		}
	}
}

func TestNamedTypeArithmeticUnwraps(t *testing.T) {
	res, err := runSource(t, namedTypeSrc+`
变量 n = 有效 加 1
type(n)
`)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := res.(*object.String); !ok || s.Value != "integer" {
		t.Fatalf("运算结果应为底层整数，实际 %v", res)
	}
}

func TestNamedTypeEquality(t *testing.T) {
	res, err := runSource(t, namedTypeSrc+`有效 等于 1`)
	if err != nil {
		t.Fatal(err)
	}
	if !isTruthy(res) {
		t.Fatal("状态值应与底层整数等值比较")
	}
}

func TestNamedTypeCastKeyword(t *testing.T) {
	res, err := runSource(t, namedTypeSrc+`
变量 s、ok = 1 转为 状态
返回 ok
`)
	if err != nil {
		t.Fatal(err)
	}
	if !isTruthy(res) {
		t.Fatal("转为 自定义类型应成功")
	}
	res2, err := runSource(t, namedTypeSrc+`
变量 s、ok = 1 转为 状态
s 是 状态
`)
	if err != nil {
		t.Fatal(err)
	}
	if !isTruthy(res2) {
		t.Fatal("转换结果应为 状态")
	}
	res3, err := runSource(t, namedTypeSrc+`
变量 s、ok = "x" 转为 状态
ok
`)
	if err != nil {
		t.Fatal(err)
	}
	if isTruthy(res3) {
		t.Fatal("无法转换应返回假")
	}
}

func TestNamedTypeMethod(t *testing.T) {
	src := namedTypeSrc + `
定义 状态 的 方法 文本(自己) -> 字符串
    判断 自己
        为 无效
            返回 "无效"
        为 有效
            返回 "有效"
    结束
    返回 "未知"
结束
有效.文本()
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := res.(*object.String); !ok || s.Value != "有效" {
		t.Fatalf("自定义类型方法错误：%v", res)
	}
}

func TestNamedTypeStringBase(t *testing.T) {
	src := `类型 姓名 字符串
变量 姓名 我 = 姓名("张三")
我 等于 "张三"
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if !isTruthy(res) {
		t.Fatal("字符串为底层类型的命名值应支持等值比较")
	}
}

func TestNamedTypeTypeOf(t *testing.T) {
	res, err := runSource(t, namedTypeSrc+`type(有效)`)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := res.(*object.String); !ok || s.Value != "状态" {
		t.Fatalf("type 应返回自定义类型名，实际 %v", res)
	}
}
