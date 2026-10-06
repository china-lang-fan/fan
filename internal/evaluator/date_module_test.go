package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupDateModule(t *testing.T) *Environment {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"系统.凡", "时间.凡", "日期.凡"} {
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

func evalDateModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupDateModule(t)
	progAST, errs := parser.ParseProgram(`导入 "日期" 作为 日期模块
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

func TestDateModuleBasic(t *testing.T) {
	src := `变量 时间戳, 错误值 = 日期模块.从年月日(2020, 1, 1)
日期模块.格式化日期(时间戳)`
	if got := evalDateModule(t, src).Inspect(); got != "2020-01-01" {
		t.Fatalf("结果为 %s，期望 2020-01-01", got)
	}
}

func TestDateModuleAge(t *testing.T) {
	src := `变量 出生, 错误一 = 日期模块.从年月日(2000, 6, 15)
变量 当前, 错误二 = 日期模块.从年月日(2020, 6, 15)
日期模块.周岁(出生, 当前)`
	if got := evalDateModule(t, src).Inspect(); got != "20" {
		t.Fatalf("结果为 %s，期望 20", got)
	}
}
