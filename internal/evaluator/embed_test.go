package evaluator

import "testing"

const embedModels = `
定义 模型 动物
    公开 变量 字符串 名字
    变量 字符串 备注
结束
定义 动物 的 方法 初始化(名字 字符串)
    自己.名字 = 名字
    自己.备注 = "默认级"
结束
定义 动物 的 方法 介绍() -> 字符串
    返回 "动物:" + 自己.名字
结束

定义 模型 狗
    嵌入 动物
    公开 变量 字符串 品种
结束
定义 狗 的 方法 初始化(名字 字符串, 品种 字符串)
    自己.名字 = 名字
    自己.品种 = 品种
    自己.备注 = "默认级"
结束
`

func TestEmbedFieldAccessAsSubInstance(t *testing.T) {
	src := embedModels + `
变量 d = 狗("旺财", "柴犬")
d.动物.介绍()
`
	mustString(t, src, "动物:旺财")
}

func TestEmbedSubInstanceIsValue(t *testing.T) {
	src := embedModels + `
变量 d = 狗("旺财", "柴犬")
变量 部分 = d.动物
部分.名字
`
	mustString(t, src, "旺财")
}

func TestEmbedPromotedFields(t *testing.T) {
	src := embedModels + `
变量 d = 狗("旺财", "柴犬")
d.名字 + "/" + d.备注
`
	mustString(t, src, "旺财/默认级")
}

func TestEmbedPromotedMethodBindsCorrectSelf(t *testing.T) {
	src := embedModels + `
变量 d = 狗("旺财", "柴犬")
d.介绍()
`
	mustString(t, src, "动物:旺财")
}

func TestEmbedQualifiedFieldWrite(t *testing.T) {
	src := embedModels + `
变量 d = 狗("旺财", "柴犬")
d.动物.名字 = "来福"
d.名字
`
	mustString(t, src, "来福")
}

func TestAmbiguousEmbeddedFieldRejected(t *testing.T) {
	src := `
定义 模型 A
    变量 整数 x
结束
定义 模型 B
    变量 整数 x
结束
定义 模型 C
    嵌入 A
    嵌入 B
结束
变量 c = C()
c.x
`
	if _, err := runSource(t, src); err == nil {
		t.Fatal("歧义字段访问应报错")
	}
}

func TestAmbiguousEmbeddedFieldWriteRejected(t *testing.T) {
	src := `
定义 模型 A
    变量 整数 x
结束
定义 模型 B
    变量 整数 x
结束
定义 模型 C
    嵌入 A
    嵌入 B
结束
变量 c = C()
c.x = 1
`
	if _, err := runSource(t, src); err == nil {
		t.Fatal("歧义字段写入应报错")
	}
}

func TestAmbiguousEmbeddedMethodRejected(t *testing.T) {
	src := `
定义 模型 A
结束
定义 A 的 方法 响()
结束
定义 模型 B
结束
定义 B 的 方法 响()
结束
定义 模型 C
    嵌入 A
    嵌入 B
结束
变量 c = C()
c.响()
`
	if _, err := runSource(t, src); err == nil {
		t.Fatal("歧义方法调用应报错")
	}
}

func TestAmbiguousResolvedByQualification(t *testing.T) {
	src := `
定义 模型 A
    变量 整数 x
结束
定义 模型 B
    变量 整数 x
结束
定义 模型 C
    嵌入 A
    嵌入 B
结束
变量 c = C()
c.A.x = 1
c.B.x = 2
c.A.x
`
	mustInt(t, src, 1)
}

const privateEmbedModels = `
定义 模型 动物
    公开 变量 字符串 名字
    私有 变量 整数 年龄
结束
定义 动物 的 方法 初始化(名字 字符串)
    自己.名字 = 名字
    自己.年龄 = 5
结束
定义 动物 的 公开 方法 年龄() -> 整数
    返回 自己.年龄
结束
定义 模型 狗
    嵌入 动物
结束
`

func TestPrivateFieldNotAccessibleThroughEmbed(t *testing.T) {
	src := `
定义 模型 动物
    公开 变量 字符串 名字
    私有 变量 整数 年龄
结束
定义 动物 的 方法 初始化(名字 字符串)
    自己.名字 = 名字
    自己.年龄 = 5
结束
定义 模型 狗
    嵌入 动物
结束
变量 d = 狗("旺财")
d.年龄
`
	if _, err := runSource(t, src); err == nil {
		t.Fatal("经嵌入访问私有字段应报错")
	}
}

func TestPrivateFieldNotAccessibleInOuterMethod(t *testing.T) {
	src := privateEmbedModels + `
定义 狗 的 方法 偷看年龄() -> 整数
    返回 自己.年龄
结束
变量 d = 狗("旺财")
d.偷看年龄()
`
	if _, err := runSource(t, src); err == nil {
		t.Fatal("外层模型方法访问嵌入的私有字段应报错")
	}
}

func TestPublicFieldAvailableThroughEmbed(t *testing.T) {
	src := privateEmbedModels + `
变量 d = 狗("旺财")
d.名字
`
	mustString(t, src, "旺财")
}

func TestPublicMethodAvailableThroughEmbed(t *testing.T) {
	src := privateEmbedModels + `
变量 d = 狗("旺财")
d.年龄()
`
	mustInt(t, src, 5)
}
