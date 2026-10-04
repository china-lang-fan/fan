package evaluator

import (
	"os"
	"path/filepath"
	"testing"

	"fan/internal/ast"
	"fan/internal/parser"
)

func TestSystemModuleWrapsNativeFunctions(t *testing.T) {
	dir := t.TempDir()
	data, err := os.ReadFile(filepath.Join("..", "..", "sdk", "系统.凡"))
	if err != nil {
		t.Fatalf("读取系统模块失败：%v", err)
	}
	systemPath := filepath.Join(dir, "系统.凡")
	if err := os.WriteFile(systemPath, data, 0600); err != nil {
		t.Fatalf("写入系统模块失败：%v", err)
	}
	src := `导入 "系统" 作为 系统
变量 结果 = []
结果 = 追加(结果, 系统.type("x"))
结果 = 追加(结果, 系统.trunc(1.8))
结果 = 追加(结果, 系统.ord("A"))
结果 = 追加(结果, 系统.char(65))
结果 = 追加(结果, 系统.len([1, 2]))
结果 = 追加(结果, 系统.append([1], 2))
结果`
	progAST, errs := parser.ParseProgram(src)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	var prog *ast.Program = progAST
	env := NewEnvironment()
	env.BaseDir = dir
	env.Loader = NewLoader(dir)
	res, err := Eval(prog, env)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res.Inspect() != `[string, 1, 65, A, 2, [1, 2]]` {
		t.Fatalf("系统模块结果错误：%s", res.Inspect())
	}
}
