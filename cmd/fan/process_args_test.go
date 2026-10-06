package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunPassesScriptArgs(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"系统.凡", "转换.凡", "进程.凡"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "sdk", name))
		if err != nil {
			t.Fatalf("读取 %s 失败：%v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatalf("写入 %s 失败：%v", name, err)
		}
	}
	path := filepath.Join(dir, "参数.凡")
	source := `导入 "进程" 作为 进程模块
导入 "系统" 作为 系统

变量 参数列表 = 进程模块.args()
系统.print(参数列表[0])
系统.print(进程模块.script())
`
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatalf("写入脚本失败：%v", err)
	}
	if err := run([]string{"run", path, "参数值"}); err != nil {
		t.Fatalf("运行脚本失败：%v", err)
	}
}
