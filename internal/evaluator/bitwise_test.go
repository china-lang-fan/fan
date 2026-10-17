package evaluator

import (
	"testing"

	"fan/internal/object"
)

func TestBitwiseWords(t *testing.T) {
	cases := map[string]int64{
		`6 位与 3`:  2,
		`6 位或 1`:  7,
		`6 异或 3`:  5,
		`1 左移 4`:  16,
		`16 右移 2`: 4,
		`位取反 0`:   -1,
	}
	for src, want := range cases {
		res, err := runSource(t, src)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		n, ok := res.(*object.Integer)
		if !ok || n.Value != want {
			t.Fatalf("%s 期望 %d，实际 %v", src, want, res)
		}
	}
}

func TestBitwiseSymbols(t *testing.T) {
	cases := map[string]int64{
		`6 & 3`:   2,
		`6 | 1`:   7,
		`6 ^ 3`:   5,
		`1 << 4`:  16,
		`16 >> 2`: 4,
		`~0`:      -1,
	}
	for src, want := range cases {
		res, err := runSource(t, src)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		n, ok := res.(*object.Integer)
		if !ok || n.Value != want {
			t.Fatalf("%s 期望 %d，实际 %v", src, want, res)
		}
	}
}

func TestBitwiseByteOperand(t *testing.T) {
	res, err := runSource(t, `变量 字节 b = 200
b 位与 15`)
	if err != nil {
		t.Fatal(err)
	}
	n, ok := res.(*object.Integer)
	if !ok || n.Value != 8 {
		t.Fatalf("字节位运算错误：%v", res)
	}
}

func TestBitwiseRejectsFloat(t *testing.T) {
	mustError(t, `1.5 位与 2`, "仅支持整数家族")
	mustError(t, `位取反 1.5`, "仅支持整数家族")
}

func TestBitwiseNegativeShift(t *testing.T) {
	mustError(t, `1 左移 -1`, "左移位数不能为负")
	mustError(t, `1 右移 -1`, "右移位数不能为负")
}

func TestBitwisePrecedence(t *testing.T) {
	res, err := runSource(t, `(1 左移 3) 位或 1 等于 9`)
	if err != nil {
		t.Fatal(err)
	}
	if !isTruthy(res) {
		t.Fatalf("位运算优先级错误：%v", res)
	}
	res, err = runSource(t, `1 左移 2 加 1`)
	if err != nil {
		t.Fatal(err)
	}
	n, ok := res.(*object.Integer)
	if !ok || n.Value != 8 {
		t.Fatalf("移位优先级应低于加法（C/Java 惯例）：%v", res)
	}
}
