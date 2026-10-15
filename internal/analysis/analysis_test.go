package analysis

import (
	"strings"
	"testing"
)

func findCode(diags []Diagnostic, code string) *Diagnostic {
	for i := range diags {
		if diags[i].Code == code {
			return &diags[i]
		}
	}
	return nil
}

func TestSyntaxErrorDiagnostic(t *testing.T) {
	diags := Analyze("函数 (\n结束\n")
	if d := findCode(diags, "syntax"); d == nil {
		t.Fatalf("期望语法错误诊断，实际：%#v", diags)
	}
	if d := findCode(diags, "syntax"); d.Severity != SeverityError || d.Line != 1 {
		t.Fatalf("语法诊断级别或位置错误：%#v", d)
	}
}

func TestMixfixSpaceWarning(t *testing.T) {
	src := `函数 修改(值 字符串)的状态为无效
    返回 值
结束

修改("x") 的状态为无效
`
	diags := Analyze(src)
	d := findCode(diags, "mixfix-space")
	if d == nil {
		t.Fatalf("期望段间空格告警，实际：%#v", diags)
	}
	if d.Severity != SeverityWarning {
		t.Fatalf("应为 warning：%#v", d)
	}
}

func TestMixfixTightNoWarning(t *testing.T) {
	src := `函数 修改(值 字符串)的状态为无效
    返回 值
结束

修改("x")的状态为无效
`
	for _, d := range Analyze(src) {
		if d.Code == "mixfix-space" || d.Code == "mixfix-unresolved" {
			t.Fatalf("紧邻调用不应告警：%#v", d)
		}
	}
}

func TestMixfixUnresolvedWarning(t *testing.T) {
	src := `函数 修改(值 字符串)的状态为有效
    返回 值
结束

修改("x")的状态为无效
`
	if d := findCode(Analyze(src), "mixfix-unresolved"); d == nil {
		t.Fatal("段名不匹配应告警")
	}
}

func TestMixfixThreeSegments(t *testing.T) {
	src := `函数 从(起点 字符串)到(终点 字符串)乘坐(工具 字符串)
    返回 起点
结束

从("北京")到("上海")乘坐("高铁")
`
	for _, d := range Analyze(src) {
		if d.Code == "mixfix-space" || d.Code == "mixfix-unresolved" {
			t.Fatalf("三段紧邻调用不应告警：%#v", d)
		}
	}

	spaced := src
	_ = spaced
	src2 := `函数 从(起点 字符串)到(终点 字符串)乘坐(工具 字符串)
    返回 起点
结束

从("北京") 到("上海")乘坐("高铁")
`
	if d := findCode(Analyze(src2), "mixfix-space"); d == nil {
		t.Fatal("直接连接段间有空格应告警")
	}
}

func TestNoWarningForPlainMemberAccess(t *testing.T) {
	src := `模型 结果
    变量 字符串 标签
结束

函数 修改(值 字符串)的状态为无效
    返回 值
结束

变量 x = 结果()
打印(x 的标签)
`
	for _, d := range Analyze(src) {
		if d.Code == "mixfix-space" {
			t.Fatalf("普通成员访问不应告警：%#v", d)
		}
	}
}

func TestUndefinedNameDiagnostic(t *testing.T) {
	src := `函数 测试(x 字符串)
    变量 y = 不存在的名字
    返回 y
结束
`
	d := findCode(Analyze(src), "undefined-name")
	if d == nil {
		t.Fatal("期望未定义名字告警")
	}
	if !strings.Contains(d.Message, "不存在的名字") {
		t.Fatalf("告警内容错误：%s", d.Message)
	}
}

func TestDefinedNameNoDiagnostic(t *testing.T) {
	src := `模型 结果
    变量 字符串 标签
结束

函数 测试()
    变量 r = 结果()
    r.标签 = "x"
    返回 r.标签
结束
`
	for _, d := range Analyze(src) {
		if d.Code == "undefined-name" {
			t.Fatalf("已定义名字不应告警：%s", d.Message)
		}
	}
}

func TestImportNotFoundDiagnostic(t *testing.T) {
	src := `导入 "肯定不存在的模块_xyz" 作为 模块
`
	d := findCode(Analyze(src), "import-not-found")
	if d == nil {
		t.Fatal("期望导入失败告警")
	}
}

func TestDocCommentExtraction(t *testing.T) {
	src := `# 修改用户状态
# 第二行说明
函数 修改(值 字符串)的状态为无效
    返回 值
结束

变量 x = 1
`
	doc := ParseDocument("file:///t.凡", src)
	syms := doc.byName["修改的状态为无效"]
	if len(syms) != 1 {
		t.Fatalf("应收集到函数符号：%v", doc.byName)
	}
	if syms[0].Doc != "修改用户状态\n第二行说明" {
		t.Fatalf("文档注释错误：%q", syms[0].Doc)
	}
}

func TestDocCommentBlankLineCuts(t *testing.T) {
	src := `# 上一段说明

函数 测试
    返回 1
结束
`
	doc := ParseDocument("file:///t.凡", src)
	if syms := doc.byName["测试"]; len(syms) == 1 && syms[0].Doc != "" {
		t.Fatalf("空行应切断文档注释，实际：%q", syms[0].Doc)
	}
}

func TestUTF16Column(t *testing.T) {
	line := "修改(\"x\") 的"
	if got := UTF16Column(line, 1); got != 0 {
		t.Fatalf("首列应为 0，实际 %d", got)
	}
	if got := UTF16Column(line, 3); got != 2 {
		t.Fatalf("BMP 中文字符列应为 2，实际 %d", got)
	}
}

func TestInterfaceSymbols(t *testing.T) {
	src := `接口 说话者
    方法 说话() -> 字符串
结束

定义 模型 狗
    变量 字符串 名
结束

定义 狗 的 方法 说话() -> 字符串
    返回 "汪"
结束

函数 介绍(谁 说话者)
    打印(谁.说话())
结束

变量 说话者 阿黄 = 狗("阿黄")
介绍(阿黄)
`
	diags := Analyze(src)
	if d := findCode(diags, "undefined-name"); d != nil {
		t.Fatalf("接口相关名字不应报未定义：%#v", d)
	}
	doc := ParseDocument("file:///t.凡", src)
	iface := doc.interfaces["说话者"]
	if iface == nil {
		t.Fatalf("未收集到接口符号")
	}
	if len(iface.Children) != 1 || iface.Children[0].Name != "说话" {
		t.Fatalf("接口方法符号错误：%#v", iface.Children)
	}
}
