package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func testdataDir() string {
	abs, _ := filepath.Abs("testdata")
	return abs
}

func moduleTestEnv() *Environment {
	dir := testdataDir()
	env := NewEnvironment()
	env.BaseDir = dir
	env.SetLoader(NewLoader(dir))
	return env
}

func runModuleSource(t *testing.T, src string) (object.Object, error) {
	t.Helper()
	prog, errs := parser.ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	return Eval(prog, moduleTestEnv())
}

var _ ast.Statement

func TestModuleImportFunction(t *testing.T) {
	src := "导入 \"lib/工具\"\n工具.加一(41)"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "42" {
		t.Fatalf("模块函数调用错误：%s", res.Inspect())
	}
}

func TestModuleImportVar(t *testing.T) {
	src := "导入 \"lib/工具\"\n工具.名"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "库" {
		t.Fatalf("模块变量读取错误：%s", res.Inspect())
	}
}

func TestModuleImportClass(t *testing.T) {
	src := "导入 \"lib/工具\"\n变量 x = 工具.项()\nx.值 = 5\nx.值"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "5" {
		t.Fatalf("模块模型实例化错误：%s", res.Inspect())
	}
}

func TestModuleImportPrimitiveMethod(t *testing.T) {
	src := "导入 \"lib/工具\"\n\"x\".库标记()"
	res, err := runModuleSource(t, src)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != "x-库" {
		t.Fatalf("模块原生方法调用错误：%s", res.Inspect())
	}
}

func TestModuleUnexportedHidden(t *testing.T) {
	src := "导入 \"lib/工具\"\n工具.私有函数()"
	_, err := runModuleSource(t, src)
	if err == nil {
		t.Fatal("未导出的函数应不可访问")
	}
}

func TestModuleNotFound(t *testing.T) {
	src := "导入 \"lib/不存在\""
	_, err := runModuleSource(t, src)
	if err == nil {
		t.Fatal("找不到模块应报错")
	}
}

func TestModuleCached(t *testing.T) {
	env := moduleTestEnv()
	m1, err := env.Loader.Load(testdataDir(), "lib/工具")
	if err != nil {
		t.Fatalf("加载失败：%v", err)
	}
	m2, err := env.Loader.Load(testdataDir(), "lib/工具")
	if err != nil {
		t.Fatalf("第二次加载失败：%v", err)
	}
	if m1 != m2 {
		t.Fatal("重复导入应返回缓存的同一模块")
	}
}

func TestModuleCircularImport(t *testing.T) {
	src := "导入 \"lib/a\""
	_, err := runModuleSource(t, src)
	if err == nil {
		t.Fatal("循环导入应报错")
	}
}

func TestModuleChineseExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "工具.凡")
	if err := os.WriteFile(path, []byte("导出 变量 名 = \"库\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(dir)
	mod, err := loader.Load(dir, "工具")
	if err != nil {
		t.Fatalf("加载失败：%v", err)
	}
	if mod.Exports["名"].Inspect() != "库" {
		t.Fatalf("模块内容错误：%s", mod.Exports["名"].Inspect())
	}
}

func TestModuleExplicitChineseExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "工具.凡")
	if err := os.WriteFile(path, []byte("导出 变量 名 = \"库\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(dir)
	mod, err := loader.Load(dir, "工具.凡")
	if err != nil {
		t.Fatalf("加载失败：%v", err)
	}
	if mod.Exports["名"].Inspect() != "库" {
		t.Fatalf("模块内容错误：%s", mod.Exports["名"].Inspect())
	}
}

func TestModuleLegacyExtensionStillSupported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "工具.fan")
	if err := os.WriteFile(path, []byte("导出 变量 名 = \"旧\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(dir)
	mod, err := loader.Load(dir, "工具")
	if err != nil {
		t.Fatalf("加载失败：%v", err)
	}
	if mod.Exports["名"].Inspect() != "旧" {
		t.Fatalf("模块内容错误：%s", mod.Exports["名"].Inspect())
	}
}
