package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupFileModule(t *testing.T) *Environment {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"系统.凡", "文件.凡"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "sdk", name))
		if err != nil {
			t.Fatalf("读取 %s 失败：%v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatalf("写入 %s 失败：%v", name, err)
		}
	}
	env := NewEnvironment()
	env.BaseDir = dir
	env.Loader = NewLoader(dir)
	return env
}

func evalFileModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupFileModule(t)
	progAST, errs := parser.ParseProgram(`导入 "文件" 作为 文件模块
` + src)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	var prog *ast.Program = progAST
	res, err := Eval(prog, env)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	return res
}

func TestFileModuleReadWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("内容"), 0600); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}
	src := `变量 内容, 错误值 = 文件模块.读取("` + path + `")
如果 错误值 != 空 那么
    返回 错误值
结束
内容`
	if got := evalFileModule(t, src).Inspect(); got != "内容" {
		t.Fatalf("读取内容为 %s，期望 内容", got)
	}
}

func TestFileModuleTempAndStat(t *testing.T) {
	src := `变量 目录, 临时错误 = 文件模块.临时目录("fan-go-")
如果 临时错误 != 空 那么
    返回 临时错误
结束
变量 创建错误 = 文件模块.写入(文件模块.拼路径(目录, "a.txt"), "x")
变量 信息, 信息错误 = 文件模块.信息(目录)
变量 删除错误 = 文件模块.删除全部(目录)
如果 创建错误 != 空 或 信息错误 != 空 或 删除错误 != 空 那么
    返回 "失败"
结束
信息["name"]`
	res := evalFileModule(t, src)
	if res.Kind() != object.KindString || res.Inspect() == "" {
		t.Fatalf("stat 结果异常：%s", res.Inspect())
	}
}
