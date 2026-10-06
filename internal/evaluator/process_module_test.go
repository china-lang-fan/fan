package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupProcessModule(t *testing.T) *Environment {
	t.Helper()
	dir := t.TempDir()
	files := []string{"系统.凡", "转换.凡", "进程.凡"}
	for _, name := range files {
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

func evalProcessModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupProcessModule(t)
	progAST, errs := parser.ParseProgram(`导入 "进程" 作为 进程模块
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

func TestProcessModulePID(t *testing.T) {
	if got := evalProcessModule(t, `进程模块.进程号()`).Inspect(); got == "0" {
		t.Fatalf("进程号不应为 0")
	}
}

func TestProcessModuleExecute(t *testing.T) {
	src := `变量 结果, 错误值 = 进程模块.运行("go", ["version"])
如果 错误值 != 空 那么
    错误值
否则
    结果["success"]
结束`
	if got := evalProcessModule(t, src).Inspect(); got != "真" {
		t.Fatalf("结果为 %s，期望 真", got)
	}
}
