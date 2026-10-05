package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupMathModule(t *testing.T) *Environment {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"系统.凡", "数学.凡"} {
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

func evalMathModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupMathModule(t)
	progAST, errs := parser.ParseProgram(`导入 "数学" 作为 数学模块
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

func TestMathModuleBasics(t *testing.T) {
	src := `数学模块.clamp(数学模块.abs(-5), 0, 3)`
	if got := evalMathModule(t, src).Inspect(); got != "3" {
		t.Fatalf("结果为 %s，期望 3", got)
	}
}

func TestMathModulePowers(t *testing.T) {
	src := `数学模块.pow(数学模块.sqrt(4), 2)`
	if got := evalMathModule(t, src).Inspect(); got != "4" {
		t.Fatalf("结果为 %s，期望 4", got)
	}
}
