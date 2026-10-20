package evaluator

import (
	"strings"
	"testing"
)

func TestUnionTypeDecl(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"变量 整数 或 字符串 a = 1\na", "1"},
		{"变量 整数 或 字符串 a = \"x\"\na", "x"},
		{"变量 整数|字符串 a = 2\na", "2"},
		{"变量 整数 | 字符串 | 布尔 a = 真\na", "真"},
		{"变量 整数 或 空 a = 空\na", "空"},
	}
	for _, c := range cases {
		res, err := runSource(t, c.src)
		if err != nil {
			t.Fatalf("%s 运行错误：%v", c.src, err)
		}
		if got := res.Inspect(); got != c.want {
			t.Errorf("%s 期望 %s，得到 %s", c.src, c.want, got)
		}
	}
}

func TestUnionTypeMismatch(t *testing.T) {
	cases := []string{
		"变量 整数 或 字符串 a = 真\na",
		"变量 整数|字符串 a = 1.5\na",
	}
	for _, src := range cases {
		_, err := runSource(t, src)
		if err == nil {
			t.Fatalf("%s 应报类型错误", src)
		}
		if !strings.Contains(err.Error(), "整数 或 字符串") {
			t.Errorf("%s 错误信息应含可读联合类型：%v", src, err)
		}
	}
}

func TestUnionTypeParamsAndReturns(t *testing.T) {
	src := `
函数 f(x 整数 或 字符串) -> 整数 或 字符串
	返回 x
结束
f(1)
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("参数/返回联合类型错误：%v", err)
	}
	if res.Inspect() != "1" {
		t.Errorf("期望 1，得到 %s", res.Inspect())
	}

	bad := `
函数 f(x 整数 或 字符串) -> 整数 或 字符串
	返回 x
结束
f(真)
`
	if _, err := runSource(t, bad); err == nil {
		t.Fatal("传入布尔应报类型不匹配")
	}

	paren := `
函数 g() -> (字符串 或 空, 整数 或 布尔)
	返回 空, 真
结束
变量 a, b = g()
b
`
	res, err = runSource(t, paren)
	if err != nil {
		t.Fatalf("多返回值联合类型错误：%v", err)
	}
	if res.Inspect() != "真" {
		t.Errorf("期望 真，得到 %s", res.Inspect())
	}
}

func TestUnionTypeByteCoercion(t *testing.T) {
	src := "变量 字节 或 字符串 a = 200\na"
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("字节联合成员错误：%v", err)
	}
	if res.Inspect() != "200" {
		t.Errorf("期望 200，得到 %s", res.Inspect())
	}
	if _, err := runSource(t, "变量 字节 或 字符串 a = 300\na"); err == nil {
		t.Fatal("字节越界应报错")
	}
}

func TestUnionTypeInterfaceMember(t *testing.T) {
	src := `
接口 说话者
	方法 说话() -> 字符串
结束
模型 人
结束
定义 人 的 方法 说话() -> 字符串
	返回 "人语"
结束
变量 说话者 或 整数 s = 人{}
s.说话()
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("接口联合成员错误：%v", err)
	}
	if res.Inspect() != "人语" {
		t.Errorf("期望 人语，得到 %s", res.Inspect())
	}
}

func TestNamedUnionTypeNewSyntax(t *testing.T) {
	src := `
定义 类型 编号 为 整数 或 字符串
变量 编号 a = 1
a
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("定义 类型 … 为 语法错误：%v", err)
	}
	if res.Inspect() != "1" {
		t.Errorf("期望 1，得到 %s", res.Inspect())
	}

	src = `
定义 类型 编号 = 整数 | 字符串
变量 编号 a = "x"
a
`
	res, err = runSource(t, src)
	if err != nil {
		t.Fatalf("定义 类型 … = 语法错误：%v", err)
	}
	if res.Inspect() != "x" {
		t.Errorf("期望 x，得到 %s", res.Inspect())
	}
}

func TestNamedUnionType(t *testing.T) {
	src := `
类型 编号 整数 或 字符串
变量 编号 a = 1
a
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("命名联合类型错误：%v", err)
	}
	if res.Inspect() != "1" {
		t.Errorf("期望 1，得到 %s", res.Inspect())
	}

	reassign := `
类型 编号 整数 或 字符串
变量 编号 a = 1
a = "A001"
a
`
	res, err = runSource(t, reassign)
	if err != nil {
		t.Fatalf("命名联合类型重赋值错误：%v", err)
	}
	if res.Inspect() != "A001" {
		t.Errorf("期望 A001，得到 %s", res.Inspect())
	}

	if _, err := runSource(t, "类型 编号 整数 或 字符串\n变量 编号 a = 真\na"); err == nil {
		t.Fatal("布尔赋给 整数 或 字符串 命名类型应报错")
	}

	convert := `
类型 编号 整数 或 字符串
变量 v, 成功 = 真 转为 编号
成功
`
	res, err = runSource(t, convert)
	if err != nil {
		t.Fatalf("转为 失败不应抛错：%v", err)
	}
	if res.Inspect() != "假" {
		t.Errorf("期望 假，得到 %s", res.Inspect())
	}

	convertOK := `
类型 编号 整数 或 字符串
变量 v = 编号(5)
v
`
	res, err = runSource(t, convertOK)
	if err != nil {
		t.Fatalf("显式构造命名联合类型错误：%v", err)
	}
	if res.Inspect() != "5" {
		t.Errorf("期望 5，得到 %s", res.Inspect())
	}

	isTest := `
类型 编号 整数 或 字符串
变量 x = 1
x 是 编号
`
	res, err = runSource(t, isTest)
	if err != nil {
		t.Fatalf("是 命名联合类型错误：%v", err)
	}
	if res.Inspect() != "真" {
		t.Errorf("期望 真，得到 %s", res.Inspect())
	}
}

func TestNamedUnionTypeModelMember(t *testing.T) {
	src := `
模型 矩形
	变量 整数 宽
结束
类型 形状 矩形 或 字符串
变量 形状 s = 矩形{宽: 3}
s.宽
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("含模型成员的命名联合类型错误：%v", err)
	}
	if res.Inspect() != "3" {
		t.Errorf("期望 3，得到 %s", res.Inspect())
	}

	method := `
模型 矩形
	变量 整数 宽
结束
类型 形状 矩形 或 字符串
变量 形状 s = "x"
s 是 字符串
`
	res, err = runSource(t, method)
	if err != nil {
		t.Fatalf("命名联合值按内部类型判断错误：%v", err)
	}
	if res.Inspect() != "真" {
		t.Errorf("期望 真，得到 %s", res.Inspect())
	}
}

func TestNamedUnionTypeAliasMethod(t *testing.T) {
	src := `
类型 编号 整数 或 字符串
定义 编号 的 方法 文本(自己) -> 字符串
	如果 自己 是 整数 那么
		返回 "整数编号"
	结束
	返回 "文本编号"
结束
变量 编号 a = "x"
a.文本()
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("命名联合类型方法错误：%v", err)
	}
	if res.Inspect() != "文本编号" {
		t.Errorf("期望 文本编号，得到 %s", res.Inspect())
	}
}

func TestNamedUnionTypeNullable(t *testing.T) {
	src := `
类型 结果 整数 或 空
变量 结果 r = 空
r
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("含空成员的命名联合类型错误：%v", err)
	}
	if res.Inspect() != "空" {
		t.Errorf("期望 空，得到 %s", res.Inspect())
	}

	assign := `
类型 结果 整数 或 空
变量 结果 r = 空
r = 3
r
`
	res, err = runSource(t, assign)
	if err != nil {
		t.Fatalf("含空成员重赋值错误：%v", err)
	}
	if res.Inspect() != "3" {
		t.Errorf("期望 3，得到 %s", res.Inspect())
	}
}

func TestUnionTypeModelField(t *testing.T) {
	src := `
模型 矩形
	变量 整数 宽
结束
模型 容器
	变量 整数 或 字符串 标签
	变量 矩形 | 字符串 内容
结束
变量 容器 c = 容器{标签: "x", 内容: 矩形{宽: 3}}
c.标签
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("模型字段联合类型错误：%v", err)
	}
	if res.Inspect() != "x" {
		t.Errorf("期望 x，得到 %s", res.Inspect())
	}
}
