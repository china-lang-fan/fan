package evaluator

import (
	"testing"

	"fan/internal/object"
)

func TestNonDecimalIntegerLiterals(t *testing.T) {
	cases := []struct {
		src  string
		want int64
	}{
		{"0b1010", 10},
		{"0B1111", 15},
		{"0o17", 15},
		{"0O777", 511},
		{"0xFF", 255},
		{"0Xff", 255},
		{"0x10", 16},
	}
	for _, c := range cases {
		res, err := runSource(t, c.src)
		if err != nil {
			t.Fatalf("%s 运行错误：%v", c.src, err)
		}
		if got := res.Inspect(); got != intText(c.want) {
			t.Errorf("%s 期望 %d，得到 %s", c.src, c.want, got)
		}
	}
}

func intText(n int64) string {
	return (&object.Integer{Value: n}).Inspect()
}
