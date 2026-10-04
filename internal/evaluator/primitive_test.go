package evaluator

import "testing"

func TestBuiltinType(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"1", "integer"},
		{"1.5", "float"},
		{`"x"`, "string"},
		{"真", "boolean"},
		{"空", "nil"},
		{"[]", "array"},
		{"{}", "dict"},
		{"错误(\"x\")", "error"},
	}
	for _, tc := range cases {
		mustString(t, "type("+tc.src+")", tc.want)
	}
	mustString(t, "函数 f()\n结束\ntype(f)", "function")
	mustString(t, "模型 C\n结束\ntype(C)", "class")
	mustString(t, "模型 C\n结束\ntype(C())", "instance")
}

func TestBuiltinTrunc(t *testing.T) {
	mustInt(t, "trunc(1.9)", 1)
	mustInt(t, "trunc(-1.9)", -1)
	mustInt(t, "trunc(2)", 2)
	mustError(t, "trunc(\"x\")", "必须是数字")
}

func TestBuiltinOrd(t *testing.T) {
	mustInt(t, `ord("A")`, 65)
	mustInt(t, `ord("凡")`, 20961)
	mustError(t, `ord("")`, "必须包含一个字符")
	mustError(t, `ord("AB")`, "必须包含一个字符")
}

func TestBuiltinChar(t *testing.T) {
	mustString(t, "char(65)", "A")
	mustString(t, "char(20961)", "凡")
	mustError(t, "char(\"x\")", "必须是整数")
}

func TestEnglishErrorBuiltin(t *testing.T) {
	res, err := runSource(t, `error("失败")`)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "错误：失败" {
		t.Fatalf("错误值不符：%s", res.Inspect())
	}
}
