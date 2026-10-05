package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupArrayModule(t *testing.T) *Environment {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"系统.凡", "字符串.凡", "数组.凡"} {
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

func evalArrayModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupArrayModule(t)
	progAST, errs := parser.ParseProgram(`导入 "数组" 作为 数组模块
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

func TestArrayModuleTransforms(t *testing.T) {
	src := `函数 加倍(值 整数) -> 整数
    返回 值 * 2
结束
函数 是偶数(值 整数) -> 布尔
    返回 值 取余 2 == 0
结束
变量 映射 = 数组模块.map([1, 2], 加倍)
变量 过滤 = 数组模块.filter([1, 2, 3, 4], 是偶数)
数组模块.flatten([映射, 过滤])`
	if got := evalArrayModule(t, src).Inspect(); got != "[2, 4, 2, 4]" {
		t.Fatalf("结果为 %s，期望 [2, 4, 2, 4]", got)
	}
}

func TestArrayModuleSorting(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`数组模块.sort([3, 1, 2])`, "[1, 2, 3]"},
		{`数组模块.sort(["c", "a", "b"])`, "[a, b, c]"},
		{`数组模块.reverse([1, 2, 3])`, "[3, 2, 1]"},
	}
	for _, tc := range cases {
		if got := evalArrayModule(t, tc.src).Inspect(); got != tc.want {
			t.Fatalf("%s => %s，期望 %s", tc.src, got, tc.want)
		}
	}
}
