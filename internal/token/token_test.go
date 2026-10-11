package token

import "testing"

func TestLookupKeyword(t *testing.T) {
	cases := []struct {
		ident string
		want  Type
	}{
		{"定义", DEFINE},
		{"变量", VAR},
		{"常量", CONST},
		{"整数", TYPE_INT},
		{"小数", TYPE_FLOAT},
		{"字符串", TYPE_STRING},
		{"布尔", TYPE_BOOL},
		{"真", TRUE},
		{"假", FALSE},
		{"空", NIL},
		{"为", ASSIGN},
		{"加等于", PLUS_EQ},
		{"减等于", MINUS_EQ},
		{"乘等于", STAR_EQ},
		{"除等于", SLASH_EQ},
		{"自增", INC},
		{"自减", DEC},
		{"加", PLUS},
		{"减", MINUS},
		{"乘", STAR},
		{"除", SLASH},
		{"取余", PERCENT},
		{"等于", EQ},
		{"不等于", NEQ},
		{"小于", LT},
		{"小于等于", LTE},
		{"大于", GT},
		{"大于等于", GTE},
		{"且", AND},
		{"或", OR},
		{"非", NOT},
		{"判断", SWITCH},
		{"其他", DEFAULT},
		{"如果", IF},
		{"那么", THEN},
		{"否则", ELSE},
		{"否则如果", ELSEIF},
		{"结束", END},
		{"当", WHILE},
		{"循环", LOOP},
		{"重复", REPEAT},
		{"直到", UNTIL},
		{"次", TIMES},
		{"跳出", BREAK},
		{"继续", CONTINUE},
		{"遍历", FORIN},
		{"中的", INOF},
		{"错误", TYPE_ERROR},
		{"字典", TYPE_DICT},
		{"稍后", DEFER},
		{"抛出", THROW},
		{"检查", CHECK},
	}
	for _, c := range cases {
		got, ok := Lookup(c.ident)
		if !ok {
			t.Errorf("关键字 %q 应被识别", c.ident)
			continue
		}
		if got != c.want {
			t.Errorf("关键字 %q 期望 %s，实际 %s", c.ident, c.want, got)
		}
	}
}

func TestLookupNonKeyword(t *testing.T) {
	for _, ident := range []string{"名字", "abc", "年龄", "注释", "hello", "捕获", "尝试"} {
		if _, ok := Lookup(ident); ok {
			t.Errorf("%q 不应被识别为语法关键字（注释由 lexer 处理）", ident)
		}
	}
}
