package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupStringModule(t *testing.T) *Environment {
	t.Helper()
	dir := t.TempDir()
	data, err := os.ReadFile(filepath.Join("..", "..", "sdk", "字符串.凡"))
	if err != nil {
		t.Fatalf("读取文本模块失败：%v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "字符串.凡"), data, 0600); err != nil {
		t.Fatalf("写入文本模块失败：%v", err)
	}
	systemData, err := os.ReadFile(filepath.Join("..", "..", "sdk", "系统.凡"))
	if err != nil {
		t.Fatalf("读取系统模块失败：%v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "系统.凡"), systemData, 0600); err != nil {
		t.Fatalf("写入系统模块失败：%v", err)
	}
	env := NewEnvironment()
	env.BaseDir = dir
	env.Loader = NewLoader(dir)
	return env
}

func evalStringModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupStringModule(t)
	progAST, errs := parser.ParseProgram(`导入 "字符串" 作为 文本模块
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

func TestStringModuleSplitAndJoin(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`文本模块.split("a,b,c", ",")`, "[a, b, c]"},
		{`文本模块.split("abc", "")`, "[a, b, c]"},
		{`文本模块.join(["a", "b", "c"], "-")`, "a-b-c"},
	}
	for _, tc := range cases {
		if got := evalStringModule(t, tc.src).Inspect(); got != tc.want {
			t.Fatalf("%s => %s，期望 %s", tc.src, got, tc.want)
		}
	}
}

func TestStringModuleSearch(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`文本模块.contains("abc", "b")`, "真"},
		{`文本模块.contains("abc", "d")`, "假"},
		{`文本模块.startsWith("abc", "a")`, "真"},
		{`文本模块.endsWith("abc", "c")`, "真"},
		{`文本模块.index("abc", "b")`, "1"},
		{`文本模块.index("abc", "d")`, "-1"},
		{`文本模块.substring("abc", 1, 2)`, "b"},
		{`文本模块.substring("abc", 1)`, "bc"},
	}
	for _, tc := range cases {
		if got := evalStringModule(t, tc.src).Inspect(); got != tc.want {
			t.Fatalf("%s => %s，期望 %s", tc.src, got, tc.want)
		}
	}
}

func TestStringModuleTransform(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`文本模块.replace("abc", "b", "x")`, "axc"},
		{`文本模块.trim("  abc  ")`, "abc"},
		{`文本模块.lower("AbC")`, "abc"},
		{`文本模块.upper("AbC")`, "ABC"},
	}
	for _, tc := range cases {
		if got := evalStringModule(t, tc.src).Inspect(); got != tc.want {
			t.Fatalf("%s => %s，期望 %s", tc.src, got, tc.want)
		}
	}
}
