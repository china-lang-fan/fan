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
	mustError(t, src, "缺少方法 读取")
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
