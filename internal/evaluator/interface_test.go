package evaluator

import (
	"testing"

	"fan/internal/object"
)

const ifaceDogSrc = `
接口 说话者
    方法 说话() -> 字符串
结束

定义 模型 狗
    变量 字符串 名
结束
定义 狗 的 方法 初始化(名 字符串)
    自己.名 = 名
结束
定义 狗 的 方法 说话() -> 字符串
    返回 "汪：" + 自己.名
结束
`

func TestInterfaceSatisfiedByModel(t *testing.T) {
	src := ifaceDogSrc + `
变量 说话者 阿黄 = 狗("阿黄")
阿黄.说话()
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := res.(*object.String); !ok || s.Value != "汪：阿黄" {
		t.Fatalf("接口调用结果错误：%v", res)
	}
}

func TestInterfaceMethodParamTypeCheck(t *testing.T) {
	src := ifaceDogSrc + `
函数 介绍(谁 说话者)
    返回 谁.说话()
结束
介绍(狗("旺财"))
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := res.(*object.String); !ok || s.Value != "汪：旺财" {
		t.Fatalf("接口参数调用错误：%v", res)
	}
}

func TestInterfaceMissingMethod(t *testing.T) {
	src := ifaceDogSrc + `
函数 介绍(谁 说话者)
结束
定义 模型 石头
结束
介绍(石头())
`
	mustError(t, src, "缺少方法 说话")
}

func TestInterfaceArityMismatch(t *testing.T) {
	src := `
接口 读取者
    方法 读取(长度 整数) -> 字符串
结束
定义 模型 假的
结束
定义 假的 的 方法 读取() -> 字符串
    返回 ""
结束
函数 用(谁 读取者)
结束
用(假的())
`
	mustError(t, src, "参数数量不符：需要 1 个，实际 0 个")
}

func TestInterfaceReturnCheck(t *testing.T) {
	src := ifaceDogSrc + `
函数 造一个() -> 说话者
    返回 42
结束
造一个()
`
	mustError(t, src, "类型不匹配")
}

func TestInterfaceEmbedding(t *testing.T) {
	src := `
接口 读者
    方法 读取(长度 整数) -> 字符串
结束
接口 写者
    方法 写入(内容 字符串)
结束
接口 读写者
    嵌入 读者
    嵌入 写者
    方法 关闭()
结束
定义 模型 内存流
    变量 字符串 缓冲
结束
定义 内存流 的 方法 初始化(初始 字符串)
    自己.缓冲 = 初始
结束
定义 内存流 的 方法 读取(长度 整数) -> 字符串
    返回 自己.缓冲
结束
定义 内存流 的 方法 写入(内容 字符串)
    自己.缓冲 = 自己.缓冲 + 内容
结束
定义 内存流 的 方法 关闭()
结束
函数 用流(流 读写者)
    流.写入("abc")
    返回 流.读取(3)
结束
用流(内存流(""))
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := res.(*object.Array); ok {
		_ = s
	}
	if s, ok := res.(*object.String); !ok || s.Value != "abc" {
		t.Fatalf("嵌入接口方法集错误：%v", res)
	}
}

func TestInterfaceEmbedsUndefined(t *testing.T) {
	src := `
接口 丙
    嵌入 不存在的
结束
`
	mustError(t, src, "未定义的接口")
}

func TestInterfaceNilAssignable(t *testing.T) {
	src := ifaceDogSrc + `
变量 说话者 谁 = 空
谁
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind() != object.KindNil {
		t.Fatalf("接口零值应为空，实际 %s", res.Kind())
	}
}

func TestInterfacePrimitiveMethodSet(t *testing.T) {
	src := `
接口 可问候
    方法 问候() -> 字符串
结束
定义 字符串 的 方法 问候() -> 字符串
    返回 "你好，" + 自己
结束
函数 打招呼(谁 可问候)
    返回 谁.问候()
结束
打招呼("凡")
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := res.(*object.String); !ok || s.Value != "你好，凡" {
		t.Fatalf("原生类型满足接口错误：%v", res)
	}
}

func TestInterfaceParamTypeMismatch(t *testing.T) {
	src := `
接口 通知者
    方法 通知(消息 字符串)
结束
定义 模型 假通知
结束
定义 假通知 的 方法 通知(消息 整数)
结束
函数 用(谁 通知者)
结束
用(假通知())
`
	mustError(t, src, "参数 消息 类型不符")
}

func TestInterfaceReturnTypeMismatch(t *testing.T) {
	src := `
接口 取名者
    方法 取名() -> 字符串
结束
定义 模型 无名氏
结束
定义 无名氏 的 方法 取名() -> 整数
    返回 1
结束
函数 用(谁 取名者)
结束
用(无名氏())
`
	mustError(t, src, "返回值类型不符")
}

func TestInterfaceUntypedImplStillSatisfies(t *testing.T) {
	src := `
接口 存储
    方法 保存(数据 字符串) -> 布尔
结束
定义 模型 内存
结束
定义 内存 的 方法 保存(数据)
    返回 真
结束
函数 用(谁 存储) -> 布尔
    返回 谁.保存("x")
结束
用(内存())
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("实现方法不标注类型时应满足接口：%v", err)
	}
	if res.Inspect() != "真" {
		t.Fatalf("结果错误：%s", res.Inspect())
	}
}

func TestInterfaceExactSignatureRequired(t *testing.T) {
	src := `
定义 模型 动物
    变量 字符串 名
结束
定义 模型 狗
    嵌入 动物
结束
接口 饲养场
    方法 领一只() -> 动物
结束
定义 模型 狗场
结束
定义 狗场 的 方法 领一只() -> 狗
    返回 狗()
结束
函数 造(场 饲养场) -> 动物
    返回 场.领一只()
结束
造(狗场())
`
	mustError(t, src, "返回值类型不符")
}

func TestInterfaceVariadicImpl(t *testing.T) {
	src := `
接口 求和器
    方法 求和(a 整数, b 整数) -> 整数
结束
定义 模型 多求和
结束
定义 多求和 的 方法 求和(首个 整数, 其余 ...整数) -> 整数
    返回 首个
结束
函数 用(器 求和器) -> 整数
    返回 器.求和(1, 2)
结束
用(多求和())
`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("可变参数实现应满足接口：%v", err)
	}
	if res.Inspect() != "1" {
		t.Fatalf("结果错误：%s", res.Inspect())
	}
}
