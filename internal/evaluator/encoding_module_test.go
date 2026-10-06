package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupEncodingModule(t *testing.T) *Environment {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"系统.凡", "编码.凡"} {
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

func evalEncodingModule(t *testing.T, src string) object.Object {
	t.Helper()
	env := setupEncodingModule(t)
	progAST, errs := parser.ParseProgram(`导入 "编码" 作为 编码模块
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

func TestEncodingModuleJSON(t *testing.T) {
	src := `变量 文本, 编码错误 = 编码模块.jsonString({"a": 1})
变量 值, 解码错误 = 编码模块.parseJSON(文本)
值["a"]`
	if got := evalEncodingModule(t, src).Inspect(); got != "1" {
		t.Fatalf("结果为 %s，期望 1", got)
	}
}

func TestEncodingModuleBase64(t *testing.T) {
	src := `变量 解码, 错误值 = 编码模块.parseBase64(编码模块.base64String("fan"))
解码`
	if got := evalEncodingModule(t, src).Inspect(); got != "fan" {
		t.Fatalf("结果为 %s，期望 fan", got)
	}
}
