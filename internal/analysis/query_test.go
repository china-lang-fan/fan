package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const querySource = `# 打招呼函数
函数 打招呼(名字 字符串) -> 字符串
    返回 "你好，" 加 名字
结束

模型 用户
    变量 字符串 名
结束

变量 当前 = 打招呼("张三")
`

func TestHoverFunction(t *testing.T) {
	idx := BuildIndex("file:///q.凡", querySource)
	// 最后一行引用“打招呼”：变量 当前 = |打招呼(...)
	hover := idx.Hover(9, 8)
	if hover == nil {
		t.Fatal("期望悬停函数")
	}
	if !contains(hover.Content, "打招呼") || !contains(hover.Content, "打招呼函数") {
		t.Fatalf("悬停内容错误：%s", hover.Content)
	}
}

func TestDefinitionFunction(t *testing.T) {
	idx := BuildIndex("file:///q.凡", querySource)
	loc := idx.Definition(9, 8)
	if loc == nil {
		t.Fatal("期望跳转到定义")
	}
	if loc.URI != "file:///q.凡" || loc.Range.Start.Line != 1 {
		t.Fatalf("定义位置错误：%+v", loc)
	}
}

func TestCompletionRoot(t *testing.T) {
	idx := BuildIndex("file:///q.凡", querySource)
	items := idx.Completion(9, 1)
	labels := map[string]bool{}
	for _, item := range items {
		labels[item.Label] = true
	}
	if !labels["打招呼"] || !labels["用户"] {
		t.Fatalf("补全缺少顶层符号：%v", labels)
	}
}

func TestCompletionClassMember(t *testing.T) {
	idx := BuildIndex("file:///q.凡", querySource)
	items := idx.memberCompletions("用户")
	found := false
	for _, item := range items {
		if item.Label == "名" {
			found = true
		}
	}
	if !found {
		t.Fatal("模型成员补全应包含字段 名")
	}
}

func TestCrossModuleHover(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir+"/助手.凡", `# 助手问候
导出 函数 问候(名字 字符串) -> 字符串
    返回 名字
结束
`)
	main := `导入 "助手" 作为 助手模块

变量 x = 助手模块.问候("a")
`
	idx := BuildIndex("file://"+dir+"/main.凡", main)
	sym := idx.lookupMember("助手模块", "问候")
	if sym == nil {
		t.Fatal("应找到跨模块函数")
	}
	if sym.Doc != "助手问候" {
		t.Fatalf("跨模块文档错误：%q", sym.Doc)
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
