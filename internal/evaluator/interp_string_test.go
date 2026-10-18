package evaluator

import "testing"

func TestInterpStringIdent(t *testing.T) {
	res, err := runSource(t, `变量 名称 = "凡语言"
"你好 $名称"`)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Inspect(); got != "你好 凡语言" {
		t.Fatalf("期望 %q，得到 %q", "你好 凡语言", got)
	}
}

func TestInterpStringExpression(t *testing.T) {
	res, err := runSource(t, `变量 版本 = 5
"v${版本 加 1}: ${2 乘 3}"`)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Inspect(); got != "v6: 6" {
		t.Fatalf("期望 %q，得到 %q", "v6: 6", got)
	}
}

func TestInterpStringNilAndMultiple(t *testing.T) {
	res, err := runSource(t, `变量 x = 空
"[$x][${x}]"`)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Inspect(); got != "[空][空]" {
		t.Fatalf("期望 %q，得到 %q", "[空][空]", got)
	}
}

func TestInterpStringLeadingTrailingAndEscapedDollar(t *testing.T) {
	res, err := runSource(t, `变量 a = 1
"$a 元 和 \$a 字面，再 $a"`)
	if err != nil {
		t.Fatal(err)
	}
	want := "1 元 和 $a 字面，再 1"
	if got := res.Inspect(); got != want {
		t.Fatalf("期望 %q，得到 %q", want, got)
	}
}

func TestInterpStringUndefinedFails(t *testing.T) {
	if _, err := runSource(t, `"值：$不存在"`); err == nil {
		t.Fatal("插值引用未声明变量应报错")
	}
}
