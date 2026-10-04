package evaluator

import (
	"bytes"
	"testing"
)

func TestBuiltinTagReplacesImplementation(t *testing.T) {
	src := `@内建("测试.加一")
定义 函数 加一(n 整数) -> 整数
    返回 空
结束
加一(41)`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "42" {
		t.Fatalf("返回值错误：%s", res.Inspect())
	}
}

func TestBuiltinTagPrintArray(t *testing.T) {
	var buf bytes.Buffer
	old := Stdout
	Stdout = &buf
	defer func() { Stdout = old }()

	src := `@内建("打印")
定义 函数 输出(内容...)
    返回 空
结束
输出([1, 2, 3])`
	if _, err := runSource(t, src); err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if buf.String() != "[1, 2, 3]\n" {
		t.Fatalf("打印输出不符：%q", buf.String())
	}
}

func TestTagTooManyPositionalArgs(t *testing.T) {
	src := `模型 标记
    变量 值
结束
@标记("一", "二")
定义 函数 测试()
结束`
	_, err := runSource(t, src)
	if err == nil {
		t.Fatal("多个未命名参数应报错")
	}
}

func TestTagMissingValueField(t *testing.T) {
	src := `模型 标记
    变量 名称
结束
@标记("值")
定义 函数 测试()
结束`
	_, err := runSource(t, src)
	if err == nil {
		t.Fatal("没有 值 字段时不应接受未命名参数")
	}
}

func TestUnknownBuiltinFunction(t *testing.T) {
	src := `@内建("不存在")
定义 函数 测试()
    返回 空
结束`
	_, err := runSource(t, src)
	if err == nil {
		t.Fatal("未注册的内建函数应报错")
	}
}

func TestBuiltinTagPrint(t *testing.T) {
	var buf bytes.Buffer
	old := Stdout
	Stdout = &buf
	defer func() { Stdout = old }()

	src := `@内建("打印")
定义 函数 输出(内容...)
    返回 空
结束
输出("你好，", "凡")`
	if _, err := runSource(t, src); err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if buf.String() != "你好， 凡\n" {
		t.Fatalf("打印输出不符：%q", buf.String())
	}
}

func TestFanTagHandler(t *testing.T) {
	src := `模型 自定义
    变量 值
结束
函数 新实现()
    返回 "来自凡语言"
结束
函数 自定义处理(上下文)
    上下文.替换实现(新实现)
结束
注册标签处理("自定义", 自定义处理)
@自定义("参数")
定义 函数 测试()
    返回 空
结束
测试()`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "来自凡语言" {
		t.Fatalf("返回值错误：%s", res.Inspect())
	}
}

func TestTagInstanceFields(t *testing.T) {
	src := `模型 路由信息
    变量 值
    变量 请求方法
结束
函数 处理路由(上下文)
    变量 结果 = 函数()
        返回 上下文.标签.请求方法 + ":" + 上下文.标签.值
    结束
    上下文.替换实现(结果)
结束
注册标签处理("路由信息", 处理路由)
@路由信息("/用户", 请求方法 = "POST")
定义 函数 创建用户()
    返回 空
结束
创建用户()`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "POST:/用户" {
		t.Fatalf("标签字段不符：%s", res.Inspect())
	}
}

func TestFanTagHandlerReadsTag(t *testing.T) {
	src := `模型 自定义
    变量 值
结束
函数 自定义处理(上下文)
    变量 结果 = 函数()
        返回 上下文.标签.值
    结束
    上下文.替换实现(结果)
结束
注册标签处理("自定义", 自定义处理)
@自定义("标签值")
定义 函数 测试()
    返回 空
结束
测试()`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "标签值" {
		t.Fatalf("返回值错误：%s", res.Inspect())
	}
}

func TestClassTagReplacesConstructor(t *testing.T) {
	src := `@内建("测试.基础实例")
模型 用户
    变量 名称
结束
变量 用户实例 = 用户()
用户实例.值`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "构造替换" {
		t.Fatalf("构造替换返回值错误：%s", res.Inspect())
	}
}

func TestMethodTagReplacesImplementation(t *testing.T) {
	src := `模型 用户
结束
@内建("测试.加一")
定义 用户 的 方法 编号(基数 整数) -> 整数
    返回 空
结束
用户().编号(41)`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "42" {
		t.Fatalf("方法标签返回值错误：%s", res.Inspect())
	}
}

func TestFanMethodHandlerReceivesInstance(t *testing.T) {
	src := `模型 自定义
    变量 值
结束
模型 用户
    变量 名称
结束
函数 新实现(实例, 基数 整数)
    如果 实例.名称 == "小明" 那么
        返回 基数
    结束
    返回 0
结束
函数 处理自定义(上下文)
    上下文.替换实现(新实现)
结束
注册标签处理("自定义", 处理自定义)
@自定义
定义 用户 的 方法 编号(基数 整数)
    返回 空
结束
变量 用户实例 = 用户()
用户实例.名称 = "小明"
用户实例.编号(42)`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "42" {
		t.Fatalf("方法返回值错误：%s", res.Inspect())
	}
}

func TestFanTagHandlerReadsCurrentCallable(t *testing.T) {
	var buf bytes.Buffer
	old := Stdout
	Stdout = &buf
	defer func() { Stdout = old }()

	src := `模型 自定义
    变量 值
结束
函数 自定义处理(上下文)
    变量 原实现 = 上下文.当前实现
    打印(原实现)
结束
注册标签处理("自定义", 自定义处理)
@自定义
定义 函数 测试()
    返回 空
结束
测试()`
	if _, err := runSource(t, src); err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if buf.String() != "<函数 测试>\n" {
		t.Fatalf("当前实现读取错误：%q", buf.String())
	}
}

func TestSecondTagReceivesReplacedCallable(t *testing.T) {
	src := `模型 第一标记
    变量 值
结束
模型 第二标记
    变量 值
结束
函数 第一处理(上下文)
    变量 新实现 = 函数(基数 整数)
        返回 基数 + 10
    结束
    上下文.替换实现(新实现)
结束
函数 第二处理(上下文)
    变量 原实现 = 上下文.当前实现
    变量 新实现 = 函数(基数 整数)
        返回 原实现(基数) + 1
    结束
    上下文.替换实现(新实现)
结束
注册标签处理("第一标记", 第一处理)
注册标签处理("第二标记", 第二处理)
@第一标记
@第二标记
定义 函数 测试(基数 整数) -> 整数
    返回 基数
结束
测试(31)`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "42" {
		t.Fatalf("连续标签返回值错误：%s", res.Inspect())
	}
}

func TestTagCreationDoesNotCallInit(t *testing.T) {
	src := `模型 标记
    变量 值
结束
定义 标记 的 方法 初始化()
    错误("不应调用初始化")
结束
@标记("测试")
定义 函数 测试()
    返回 通过
结束
变量 通过 = 真
测试()`
	mustBool(t, src, true)
}

func TestTagWithoutHandlerHasNoEffect(t *testing.T) {
	src := `模型 标记
结束
@标记
定义 函数 测试()
    返回 42
结束
测试()`
	res, err := runSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "42" {
		t.Fatalf("返回值错误：%s", res.Inspect())
	}
}
