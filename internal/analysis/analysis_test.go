package analysis

import "testing"

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

func TestUTF16Column(t *testing.T) {
	line := "修改(\"x\") 的"
	if got := UTF16Column(line, 1); got != 0 {
		t.Fatalf("首列应为 0，实际 %d", got)
	}
	if got := UTF16Column(line, 3); got != 2 {
		t.Fatalf("BMP 中文字符列应为 2，实际 %d", got)
	}
}
