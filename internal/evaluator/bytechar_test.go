package evaluator

import (
	"testing"

	"fan/internal/object"
	"fan/internal/parser"
)

func TestCharLiteral(t *testing.T) {
	res, err := runSource(t, `'A'`)
	if err != nil {
		t.Fatal(err)
	}
	c, ok := res.(*object.Char)
	if !ok {
		t.Fatalf("期望字符，实际 %T", res)
	}
	if c.Value != 'A' {
		t.Fatalf("字符错误：%q", c.Value)
	}
}

func TestChineseCharLiteral(t *testing.T) {
	res, err := runSource(t, `'中'`)
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := res.(*object.Char); !ok || c.Value != '中' {
		t.Fatalf("中文字符错误：%v", res)
	}
}

func TestFullWidthCharLiteral(t *testing.T) {
	for _, src := range []string{"‘A’", "‘中’", "‘\\n’"} {
		res, err := runSource(t, "变量 字符 c = "+src+"\nc")
		if err != nil {
			t.Fatalf("%s 运行错误：%v", src, err)
		}
		if _, ok := res.(*object.Char); !ok {
			t.Fatalf("%s 期望字符，实际 %T", src, res)
		}
	}
	res, err := runSource(t, "‘凡’")
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := res.(*object.Char); !ok || c.Value != '凡' {
		t.Fatalf("全角字符错误：%v", res)
	}
}

func TestEscapedCharLiteral(t *testing.T) {
	cases := []struct {
		src  string
		want rune
	}{
		{`'\n'`, '\n'},
		{`'\t'`, '\t'},
		{`'\\'`, '\\'},
	}
	for _, tc := range cases {
		src, want := tc.src, tc.want
		res, err := runSource(t, src)
		if err != nil {
			t.Fatalf("%s 运行错误：%v", src, err)
		}
		if c, ok := res.(*object.Char); !ok || c.Value != want {
			t.Fatalf("%s 期望 %q，实际 %v", src, want, res)
		}
	}
}

func TestCharDeclType(t *testing.T) {
	res, err := runSource(t, `变量 字符 c = '凡'
c`)
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := res.(*object.Char); !ok || c.Value != '凡' {
		t.Fatalf("字符变量错误：%v", res)
	}
	_, err = runSource(t, `变量 字符 c = "字符串"`)
	if err == nil {
		t.Fatal("字符串不能赋给字符")
	}
}

func TestByteDeclRange(t *testing.T) {
	res, err := runSource(t, `变量 字节 b = 200
b`)
	if err != nil {
		t.Fatal(err)
	}
	if b, ok := res.(*object.Byte); !ok || b.Value != 200 {
		t.Fatalf("字节变量错误：%v", res)
	}
	if _, err := runSource(t, `变量 字节 b = 256`); err == nil {
		t.Fatal("256 超出字节范围应报错")
	}
	if _, err := runSource(t, `变量 字节 b = -1`); err == nil {
		t.Fatal("-1 超出字节范围应报错")
	}
}

func TestCharArithmeticAndCompare(t *testing.T) {
	mustInspect(t, `'A' 等于 'A'`, "真")
	mustInspect(t, `'A' 小于 'B'`, "真")
	mustInspect(t, `'中' 大于 'A'`, "真")

	res, err := runSource(t, `'A' + 1`)
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := res.(*object.Char); !ok || c.Value != 'B' {
		t.Fatalf("字符加整数应为字符，实际 %v", res)
	}
}

func TestCharStringConcat(t *testing.T) {
	mustInspect(t, `"a" + 'b'`, "ab")
	mustInspect(t, `'a' + "b"`, "ab")
}

func TestByteArithmetic(t *testing.T) {
	src := `
变量 字节 a = 10
变量 字节 b = 20
a + b`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if i, ok := res.(*object.Integer); !ok || i.Value != 30 {
		t.Fatalf("字节算术错误：%v", res)
	}
}

func TestCharTypeName(t *testing.T) {
	if got := runtimeTypeName(&object.Char{Value: 'x'}); got != "char" {
		t.Fatalf("字符类型名错误：%s", got)
	}
	if got := runtimeTypeName(&object.Byte{Value: 1}); got != "byte" {
		t.Fatalf("字节类型名错误：%s", got)
	}
}

func TestNativeToByte(t *testing.T) {
	rt := NewRuntime()
	vals, err := nativeToByte(rt, []object.Object{&object.Integer{Value: 200}})
	if err != nil {
		t.Fatal(err)
	}
	if b, ok := vals[0].(*object.Byte); !ok || b.Value != 200 {
		t.Fatalf("转字节错误：%v", vals[0])
	}
	vals, err = nativeToByte(rt, []object.Object{&object.Integer{Value: 256}})
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := vals[1].(*object.Error); !ok {
		t.Fatalf("超范围应返回错误值，实际 %v", vals)
	} else if e.Message == "" {
		t.Fatal("错误消息不应为空")
	}
}

func TestCharLexErrors(t *testing.T) {
	for _, src := range []string{"'ab'", "'", "'\n'"} {
		prog, errs := parser.ParseProgram(src)
		if len(errs) == 0 && prog != nil {
			t.Fatalf("%q 应产生词法/解析错误", src)
		}
	}
}

func TestNativeToChar(t *testing.T) {
	rt := NewRuntime()
	vals, err := nativeToChar(rt, []object.Object{&object.Integer{Value: 65}})
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := vals[0].(*object.Char); !ok || c.Value != 'A' {
		t.Fatalf("整数转字符错误：%v", vals[0])
	}
	vals, err = nativeToChar(rt, []object.Object{&object.String{Value: "凡"}})
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := vals[0].(*object.Char); !ok || c.Value != '凡' {
		t.Fatalf("单字符串转字符错误：%v", vals[0])
	}
	if vals, err = nativeToChar(rt, []object.Object{&object.String{Value: "多个字"}}); err != nil {
		t.Fatal(err)
	} else if _, ok := vals[1].(*object.Error); !ok {
		t.Fatalf("多字符字符串应返回错误，实际 %v", vals)
	}
}
