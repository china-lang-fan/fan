package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupRandomModule(t *testing.T) *Environment {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"系统.凡", "字符串.凡", "数组.凡", "随机.凡"} {
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

func evalRandomModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupRandomModule(t)
	progAST, errs := parser.ParseProgram(`导入 "随机" 作为 随机模块
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

func TestRandomModuleBounds(t *testing.T) {
	src := `变量 值 = 随机模块.int(5)
值 >= 0 且 值 < 5`
	if got := evalRandomModule(t, src).Inspect(); got != "真" {
		t.Fatalf("结果为 %s，期望 真", got)
	}
}

func TestRandomModuleSample(t *testing.T) {
	src := `长度(随机模块.sample([1, 2, 3], 2))`
	if got := evalRandomModule(t, src).Inspect(); got != "2" {
		t.Fatalf("结果为 %s，期望 2", got)
	}
}
