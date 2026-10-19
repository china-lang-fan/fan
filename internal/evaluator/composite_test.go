package evaluator

import "testing"

const compositeClassSrc = `
定义 模型 矩形
    公开 变量 整数 宽
    公开 变量 整数 高
    私有 变量 整数 密
结束
定义 模型 基础
    公开 变量 整数 编号
结束
定义 模型 画布
    公开 变量 字符串 标题
    嵌入 矩形
    嵌入 基础
结束
`

func TestCompositeLiteralFields(t *testing.T) {
	res, err := runSource(t, compositeClassSrc+`
变量 r = 矩形{宽: 3, 高: 4}
[r.宽, r.高]`)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Inspect(); got != "[3, 4]" {
		t.Fatalf("期望 [3, 4]，得到 %s", got)
	}
}

func TestCompositeLiteralZeroAndSubset(t *testing.T) {
	res, err := runSource(t, compositeClassSrc+`
变量 r = 矩形{宽: 9}
[r.宽, r.高]`)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Inspect(); got != "[9, 空]" {
		t.Fatalf("未列字段应为零值，得到 %s", got)
	}
}

func TestCompositeLiteralDoesNotCallInit(t *testing.T) {
	res, err := runSource(t, `
定义 模型 矩形
    公开 变量 整数 宽
结束
定义 矩形 的 方法 初始化(宽 整数)
    自己.宽 = 宽 乘 100
结束
矩形{宽: 3}.宽`)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Inspect(); got != "3" {
		t.Fatalf("复合字面量不应调用初始化，得到 %s", got)
	}
}

func TestCompositeLiteralUnknownField(t *testing.T) {
	if _, err := runSource(t, compositeClassSrc+`矩形{宽: 1, 不存在: 2}`); err == nil {
		t.Fatal("未知字段应报错")
	}
}

func TestCompositeLiteralPrivateField(t *testing.T) {
	if _, err := runSource(t, compositeClassSrc+`矩形{密: 1}`); err == nil {
		t.Fatal("私有字段不应能通过复合字面量设置")
	}
}

func TestCompositeLiteralEmbedWhole(t *testing.T) {
	res, err := runSource(t, compositeClassSrc+`
变量 c = 画布{标题: "t", 矩形: 矩形{宽: 1, 高: 2}}
[c.标题, c.矩形.宽, c.矩形.高]`)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Inspect(); got != "[t, 1, 2]" {
		t.Fatalf("嵌入整体赋值错误：%s", got)
	}
}

func TestCompositeLiteralEmbedWrongType(t *testing.T) {
	if _, err := runSource(t, compositeClassSrc+`
画布{标题: "t", 矩形: 基础{编号: 1}}`); err == nil {
		t.Fatal("嵌入字段赋错模型类型应报错")
	}
}

func TestCompositeLiteralMethodChain(t *testing.T) {
	res, err := runSource(t, `
定义 模型 矩形
    公开 变量 整数 宽
结束
定义 矩形 的 方法 翻倍() -> 整数
    返回 自己.宽 乘 2
结束
矩形{宽: 21}.翻倍()`)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Inspect(); got != "42" {
		t.Fatalf("复合字面量方法链错误：%s", got)
	}
}

func TestCompositeLiteralCrossModule(t *testing.T) {
	dir := writeTempModule(t, "图形", `
公开 定义 模型 矩形
    公开 变量 整数 宽
    公开 变量 整数 高
结束
`)
	src := `导入 "` + dir + `/图形" 作为 图形
变量 r = 图形.矩形{宽: 3, 高: 4}
[r.宽, r.高]`
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Inspect(); got != "[3, 4]" {
		t.Fatalf("跨模块复合字面量错误：%s", got)
	}
}
