package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupConversionModule(t *testing.T) *Environment {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"系统.凡", "转换.凡"} {
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

func evalConversionModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupConversionModule(t)
	progAST, errs := parser.ParseProgram(`导入 "转换" 作为 转换模块
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

func TestConversionModuleBasic(t *testing.T) {
	src := `转换模块.toString(转换模块.toFloat("3.5"))`
	if got := evalConversionModule(t, src).Inspect(); got != "3.5" {
		t.Fatalf("结果为 %s，期望 3.5", got)
	}
}

func TestConversionModuleParseError(t *testing.T) {
	src := `变量 值, 错误值 = 转换模块.parseStringAsInt("bad")
错误值 != 空`
	if got := evalConversionModule(t, src).Inspect(); got != "真" {
		t.Fatalf("结果为 %s，期望 真", got)
	}
}
