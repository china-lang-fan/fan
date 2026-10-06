package evaluator

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

func setupNetworkModule(t *testing.T) (*Environment, string) {
	t.Helper()
	dir := t.TempDir()
	files := []string{"系统.凡", "字典.凡", "文件.凡", "编码.凡", "转换.凡", "网络.凡"}
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
	return env, dir
}

func evalNetworkScript(t *testing.T, src string) object.Object {
	t.Helper()
	env, _ := setupNetworkModule(t)
	progAST, errs := parser.ParseProgram(`导入 "网络" 作为 网络模块
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

func TestNetworkModuleGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("方法为 %s，期望 GET", r.Method)
		}
		if r.URL.Query().Get("name") != "凡" {
			t.Fatalf("查询参数不正确：%s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	src := `变量 结果, 错误值 = 网络模块.获取("` + server.URL + `?name=%E5%87%A1", {"timeout": 2})
如果 错误值 != 空 那么
    错误值
否则
    结果["body"]
结束`
	if got := evalNetworkScript(t, src).Inspect(); got != `{"ok":true}` {
		t.Fatalf("结果为 %s，期望 {\"ok\":true}", got)
	}
}

func TestNetworkModulePostJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("方法为 %s，期望 POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("Content-Type 为 %s", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"received":true}`))
	}))
	defer server.Close()

	src := `变量 值, 错误值 = 网络模块.提交JSON("` + server.URL + `", {"name": "凡"}, {"timeout": 2})
如果 错误值 != 空 那么
    错误值
否则
    值["received"]
结束`
	if got := evalNetworkScript(t, src).Inspect(); got != "真" {
		t.Fatalf("结果为 %s，期望 真", got)
	}
}

func TestNetworkModuleDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("文件内容"))
	}))
	defer server.Close()
	env, dir := setupNetworkModule(t)
	target := filepath.Join(dir, "下载.txt")
	progAST, errs := parser.ParseProgram(`导入 "网络" 作为 网络模块
网络模块.下载("` + server.URL + `", "` + target + `", {"timeout": 2})`)
	if len(errs) > 0 {
		t.Fatalf("解析失败：%v", errs)
	}
	var prog *ast.Program = progAST
	res, err := Eval(prog, env)
	if err != nil {
		t.Fatalf("运行错误：%v", err)
	}
	if res != object.Null {
		t.Fatalf("下载错误：%s", res.Inspect())
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("读取下载文件失败：%v", err)
	}
	if string(data) != "文件内容" {
		t.Fatalf("内容为 %s，期望 文件内容", string(data))
	}
}
