package testrun

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fan/internal/ast"
	"fan/internal/parser"
)

func copySDKFile(t *testing.T, dir, name string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "sdk", name))
	if err != nil {
		t.Fatalf("读取 %s 失败：%v", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
		t.Fatalf("写入 %s 失败：%v", name, err)
	}
}

func TestRunnerPassingAndFailing(t *testing.T) {
	dir := t.TempDir()
	copySDKFile(t, dir, "系统.凡")
	copySDKFile(t, dir, "测试.凡")
	src := `导入 "测试" 作为 断言模块

函数 测试通过()
    断言模块.断言相等(1, 1)
结束

函数 测试失败()
    断言模块.断言相等(1, 2)
结束
`
	if err := os.WriteFile(filepath.Join(dir, "示例_测试.凡"), []byte(src), 0600); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}
	var out bytes.Buffer
	results, err := Runner{Out: &out}.Run(dir)
	if err != nil {
		t.Fatalf("运行测试失败：%v", err)
	}
	if len(results) != 1 {
		t.Fatalf("结果数量为 %d，期望 1", len(results))
	}
	if len(results[0].Tests) != 2 {
		t.Fatalf("测试数量为 %d，期望 2", len(results[0].Tests))
	}
	if len(results[0].Failures) != 1 {
		t.Fatalf("失败数量为 %d，期望 1", len(results[0].Failures))
	}
	if results[0].Failures[0].Name != "测试失败" {
		t.Fatalf("失败用例为 %s，期望 测试失败", results[0].Failures[0].Name)
	}
	if !strings.Contains(out.String(), "通过 测试通过") || !strings.Contains(out.String(), "失败 测试失败") {
		t.Fatalf("输出不符合期望：%s", out.String())
	}
}

func TestRunnerTaggedTest(t *testing.T) {
	dir := t.TempDir()
	copySDKFile(t, dir, "系统.凡")
	copySDKFile(t, dir, "测试.凡")
	src := `导入 "测试" 作为 断言模块

@测试
函数 标签用例()
    断言模块.断言为真(真)
结束
`
	if err := os.WriteFile(filepath.Join(dir, "标签_测试.凡"), []byte(src), 0600); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}
	progAST, parseErrs := parser.ParseProgram(src)
	t.Logf("parse errors: %v", parseErrs)
	for i, stmt := range progAST.Statements {
		t.Logf("stmt %d: %T %s", i, stmt, stmt.String())
		if decl, ok := stmt.(*ast.VarDecl); ok {
			if lit, ok := decl.Value.(*ast.FunctionLiteral); ok {
				t.Logf("stmt %d: %s tags %d", i, decl.Name, len(lit.Tags))
			}
		}
	}
	results, err := Runner{Out: &bytes.Buffer{}}.Run(dir)
	if err != nil {
		t.Fatalf("运行测试失败：%v", err)
	}
	if len(results) != 1 || results[0].SetupError != "" {
		t.Fatalf("setup failed: %+v", results)
	}
	if len(results[0].Tests) != 1 || results[0].Tests[0].Name != "标签用例" {
		t.Fatalf("未识别标签测试：%+v", results[0].Tests)
	}
	if results[0].Failed() {
		t.Fatalf("标签测试不应失败：%+v", results[0].Failures)
	}
}

func TestRunnerSetupError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "坏_测试.凡"), []byte("函数 测试坏("), 0600); err != nil {
		t.Fatalf("写入测试文件失败：%v", err)
	}
	results, err := Runner{Out: &bytes.Buffer{}}.Run(dir)
	if err != nil {
		t.Fatalf("运行测试失败：%v", err)
	}
	if !results[0].Failed() || results[0].SetupError == "" {
		t.Fatalf("期望初始化失败，实际 %+v", results[0])
	}
}
