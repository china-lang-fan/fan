package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"fan/internal/object"
)

const privateFieldModel = `
定义 模型 账户
    公开 变量 字符串 名字
    私有 变量 整数 余额
结束

定义 账户 的 方法 初始化(名字 字符串, 余额 整数)
    自己.名字 = 名字
    自己.余额 = 余额
结束

定义 账户 的 公开 方法 存款(数额 整数)
    自己.余额 = 自己.余额 加 数额
结束

定义 账户 的 私有 方法 内部密语() -> 字符串
    返回 "密-" + 自己.名字
结束

定义 账户 的 公开 方法 密语() -> 字符串
    返回 自己.内部密语()
结束
`

func TestPrivateFieldAccessibleInsideMethod(t *testing.T) {
	src := privateFieldModel + `
变量 a = 账户("张三", 100)
a.存款(50)
a.密语()
`
	mustString(t, src, "密-张三")
}

func TestPrivateFieldDeniedOutside(t *testing.T) {
	src := privateFieldModel + `
变量 a = 账户("张三", 100)
a.余额
`
	if _, err := runSource(t, src); err == nil {
		t.Fatal("模型外访问私有字段应报错")
	}
}

func TestPrivateFieldAssignDeniedOutside(t *testing.T) {
	src := privateFieldModel + `
变量 a = 账户("张三", 100)
a.余额 = 5
`
	if _, err := runSource(t, src); err == nil {
		t.Fatal("模型外修改私有字段应报错")
	}
}

func TestPrivateMethodDeniedOutside(t *testing.T) {
	src := privateFieldModel + `
变量 a = 账户("张三", 100)
a.内部密语()
`
	if _, err := runSource(t, src); err == nil {
		t.Fatal("模型外调用私有方法应报错")
	}
}

func TestPublicFieldAccessibleOutside(t *testing.T) {
	src := privateFieldModel + `
变量 a = 账户("张三", 100)
a.名字
`
	mustString(t, src, "张三")
}

func TestPublicMethodCallSyntax(t *testing.T) {
	src := privateFieldModel + `
变量 a = 账户("张三", 100)
a.存款(10)
a.密语()
`
	mustString(t, src, "密-张三")
}

func TestDefaultFieldVisibilityFile(t *testing.T) {
	src := rectModel + "\n变量 r = 矩形(2, 3)\nr.宽"
	mustInt(t, src, 2)
}

func loadTempModule(t *testing.T, src string) *object.Module {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "临时库.凡")
	if err := os.WriteFile(path, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(dir)
	mod, err := loader.Load(dir, "临时库")
	if err != nil {
		t.Fatalf("加载失败：%v", err)
	}
	return mod
}

func TestModuleExportsOnlyPublic(t *testing.T) {
	src := `
公开 函数 加一(x 整数) -> 整数
    返回 x 加 1
结束

函数 内部助手()
    返回 1
结束
`
	mod := loadTempModule(t, src)
	if _, ok := mod.Exports["加一"]; !ok {
		t.Fatal("公开函数应被导出")
	}
	if _, ok := mod.Exports["内部助手"]; ok {
		t.Fatal("未标记公开的函数不应被导出")
	}
}

func TestPrivateKeywordIsNotExported(t *testing.T) {
	src := `
私有 函数 内部()
    返回 1
结束
`
	mod := loadTempModule(t, src)
	if _, ok := mod.Exports["内部"]; ok {
		t.Fatal("私有函数不应被导出")
	}
}
