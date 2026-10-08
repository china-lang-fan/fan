package parser

import (
	"strings"
	"testing"

	"fan/internal/ast"
)

func TestFunctionDefinitionWithParens(t *testing.T) {
	src := strings.Join([]string{
		`函数 求和（甲，乙）`,
		`    返回 甲 加 乙`,
		`结束`,
	}, "\n")
	prog := mustParse(t, src)
	decl := prog.Statements[0].(*ast.VarDecl)
	fn, ok := decl.Value.(*ast.FunctionLiteral)
	if !ok {
		t.Fatalf("应为函数字面量，实际 %T", decl.Value)
	}
	if len(fn.Params) != 2 || fn.Params[0].Name != "甲" || fn.Params[1].Name != "乙" {
		t.Fatalf("参数解析错误：%v", fn.Params)
	}
	if fn.Name != "求和" {
		t.Fatalf("函数名应为 求和，实际 %s", fn.Name)
	}
}

func TestAnonymousFunction(t *testing.T) {
	src := strings.Join([]string{
		`变量 平方 = 函数（数）`,
		`    返回 数 乘 数`,
		`结束`,
	}, "\n")
	prog := mustParse(t, src)
	decl := prog.Statements[0].(*ast.VarDecl)
	fn, ok := decl.Value.(*ast.FunctionLiteral)
	if !ok {
		t.Fatalf("应为匿名函数，实际 %T", decl.Value)
	}
	if fn.Name != "" {
		t.Fatalf("匿名函数名应为空，实际 %s", fn.Name)
	}
	if len(fn.Params) != 1 || fn.Params[0].Name != "数" {
		t.Fatalf("参数解析错误：%v", fn.Params)
	}
}

func TestReturnNoValue(t *testing.T) {
	prog := mustParse(t, "返回")
	if _, ok := prog.Statements[0].(*ast.ReturnStmt); !ok {
		t.Fatalf("应为 返回 语句，实际 %T", prog.Statements[0])
	}
}

func TestNormalParenCall(t *testing.T) {
	prog := mustParse(t, `打印("你好")`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.CallExpr)
	if id, ok := call.Callee.(*ast.Identifier); !ok || id.Name != "打印" {
		t.Fatalf("调用方错误：%s", call.Callee.String())
	}
	if len(call.Args) != 1 {
		t.Fatalf("应有 1 个参数，实际 %d", len(call.Args))
	}
}

func TestImplicitCallNormal(t *testing.T) {
	prog := mustParse(t, `打印 "你好"`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.CallExpr)
	if id, ok := call.Callee.(*ast.Identifier); !ok || id.Name != "打印" {
		t.Fatalf("调用方错误：%s", call.Callee.String())
	}
	if len(call.Args) != 1 {
		t.Fatalf("应有 1 个参数，实际 %d", len(call.Args))
	}
}

func TestImplicitReceiverCall(t *testing.T) {
	prog := mustParse(t, `名单 追加 "张三"`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.ImplicitReceiverCallExpr)
	if call.Name != "追加" {
		t.Fatalf("方法名应为 追加，实际 %s", call.String())
	}
	if call.Receiver.String() != "名单" || len(call.Args) != 1 {
		t.Fatalf("接收者和参数解析错误，实际 %s", call.String())
	}
}

func TestLiteralImplicitReceiverCall(t *testing.T) {
	prog := mustParse(t, `"a,b,c" 拆分(",")`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.ImplicitReceiverCallExpr)
	if call.Name != "拆分" {
		t.Fatalf("方法名错误：%s", call.String())
	}
	if _, ok := call.Receiver.(*ast.StringLiteral); !ok || len(call.Args) != 1 {
		t.Fatalf("字面量接收者解析错误：%s", call.String())
	}
}

func TestLiteralImplicitReceiverNoArgCall(t *testing.T) {
	prog := mustParse(t, `"abc" 转大写`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.ImplicitReceiverCallExpr)
	if call.Name != "转大写" || len(call.Args) != 0 {
		t.Fatalf("无参接收者方法解析错误：%s", call.String())
	}
}

func TestLiteralImplicitReceiverChainedCall(t *testing.T) {
	prog := mustParse(t, `"a,b,c" 拆分(",") 反转`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.ImplicitReceiverCallExpr)
	if call.Name != "反转" || len(call.Args) != 0 {
		t.Fatalf("链式方法解析错误：%s", call.String())
	}
	if _, ok := call.Receiver.(*ast.ImplicitReceiverCallExpr); !ok {
		t.Fatalf("前一方法调用解析错误：%s", call.String())
	}
}

func TestMemberNoArgCall(t *testing.T) {
	prog := mustParse(t, "名单 的 长度")
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	if _, ok := stmt.Expression.(*ast.MaybeCallExpr); !ok {
		t.Fatalf("应为可能调用表达式，实际 %T", stmt.Expression)
	}
}

func TestMemberMethodCall(t *testing.T) {
	prog := mustParse(t, `名单 的 追加("张三")`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.CallExpr)
	if _, ok := call.Callee.(*ast.MemberExpr); !ok {
		t.Fatalf("应为成员方法调用，实际 %T", stmt.Expression)
	}
}

func TestImplicitCallMultipleArgsWithParens(t *testing.T) {
	prog := mustParse(t, `追加(名单, "张三")`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.CallExpr)
	if call.Callee.(*ast.Identifier).Name != "追加" {
		t.Fatalf("调用方错误")
	}
	if len(call.Args) != 2 {
		t.Fatalf("应有 2 个参数，实际 %d", len(call.Args))
	}
}

func TestImplicitReceiverCallWithMultipleArgs(t *testing.T) {
	prog := mustParse(t, `名单 追加 "张三", 18`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.ImplicitReceiverCallExpr)
	if call.Name != "追加" {
		t.Fatalf("方法名应为 追加，实际 %s", call.String())
	}
	if len(call.Args) != 2 {
		t.Fatalf("应有 2 个参数，实际 %d", len(call.Args))
	}
}

func TestImplicitCallNestedInArg(t *testing.T) {
	prog := mustParse(t, `打印 名单 的 长度`)
	stmt := prog.Statements[0].(*ast.ExpressionStmt)
	call := stmt.Expression.(*ast.CallExpr)
	if call.Callee.(*ast.Identifier).Name != "打印" {
		t.Fatalf("外层应为 打印，实际 %s", call.Callee.String())
	}
	if _, ok := call.Args[0].(*ast.MemberExpr); !ok {
		t.Fatalf("参数应为成员访问，实际 %s", call.Args[0].String())
	}
}

func TestNoImplicitInCondition(t *testing.T) {
	src := strings.Join([]string{
		`变量 甲 = 1`,
		`如果 甲 加 1 等于 2 那么`,
		`    甲 = 2`,
		`结束`,
	}, "\n")
	prog := mustParse(t, src)
	ifStmt := prog.Statements[1].(*ast.IfStmt)
	if _, ok := ifStmt.Branches[0].Condition.(*ast.BinaryExpr); !ok {
		t.Fatalf("条件应为二元表达式，实际 %T", ifStmt.Branches[0].Condition)
	}
}

func TestDefineFunctionOptional(t *testing.T) {
	src := strings.Join([]string{
		`定义 函数 求和(甲, 乙)`,
		`    返回 甲 加 乙`,
		`结束`,
	}, "\n")
	prog := mustParse(t, src)
	decl := prog.Statements[0].(*ast.VarDecl)
	fn := decl.Value.(*ast.FunctionLiteral)
	if fn.Name != "求和" {
		t.Fatalf("函数名应为 求和，实际 %s", fn.Name)
	}
}

func TestTypedParameters(t *testing.T) {
	src := `定义 函数 求和(甲 整数，乙 整数) -> 整数
    返回 甲 加 乙
结束`
	prog := mustParse(t, src)
	fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
	if len(fn.Params) != 2 {
		t.Fatalf("应有 2 个参数，实际 %d", len(fn.Params))
	}
	if fn.Params[0].Type != ast.TypeInt || fn.Params[0].Name != "甲" {
		t.Fatalf("首个参数错误：%+v", fn.Params[0])
	}
	if fn.Params[1].Type != ast.TypeInt || fn.Params[1].Name != "乙" {
		t.Fatalf("第二个参数错误：%+v", fn.Params[1])
	}
	if len(fn.ReturnTypes) != 1 || fn.ReturnTypes[0] != ast.TypeInt {
		t.Fatalf("返回类型错误：%+v", fn.ReturnTypes)
	}
}

func TestDefaultAndVariadicParameters(t *testing.T) {
	src := `函数 f(a 整数, b 整数 = 2, 其余 ...整数)
结束`
	prog := mustParse(t, src)
	fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
	if len(fn.Params) != 3 {
		t.Fatalf("应有 3 个参数，实际 %d", len(fn.Params))
	}
	if fn.Params[1].Default == nil {
		t.Fatalf("第二个参数应有默认值")
	}
	if !fn.Params[2].Variadic {
		t.Fatalf("第三个参数应是可变参数")
	}
}

func TestInvalidParameterOrder(t *testing.T) {
	expectError(t, "函数 f(a ...整数, b)\n结束")
	expectError(t, "函数 f(a = 1, b)\n结束")
}

func TestTypedParamsMultiReturn(t *testing.T) {
	src := "函数 商余(a 整数，b 整数) -> (整数，整数)\n    返回 a / b，a % b\n结束"
	prog := mustParse(t, src)
	fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
	if len(fn.ReturnTypes) != 2 ||
		fn.ReturnTypes[0] != ast.TypeInt || fn.ReturnTypes[1] != ast.TypeInt {
		t.Fatalf("返回类型解析错误：%+v", fn.ReturnTypes)
	}
	ret := fn.Body.Statements[0].(*ast.ReturnStmt)
	if len(ret.Values) != 2 {
		t.Fatalf("返回值数量应为 2，实际 %d", len(ret.Values))
	}
}

func TestMultiDecl(t *testing.T) {
	src := "变量 a、b = 商余(10, 3)"
	prog := mustParse(t, src)
	decl := prog.Statements[0].(*ast.MultiDecl)
	if len(decl.Names) != 2 || decl.Names[0] != "a" || decl.Names[1] != "b" {
		t.Fatalf("多变量声明解析错误：%+v", decl.Names)
	}
}

func TestMultiAssign(t *testing.T) {
	src := "a、b = 商余(10, 3)"
	prog := mustParse(t, src)
	assign := prog.Statements[0].(*ast.MultiAssign)
	if len(assign.Names) != 2 {
		t.Fatalf("多目标赋值解析错误：%+v", assign.Names)
	}
}

func TestCheckExpression(t *testing.T) {
	src := "变量 x = 检查 f()"
	prog := mustParse(t, src)
	decl := prog.Statements[0].(*ast.VarDecl)
	if _, ok := decl.Value.(*ast.CheckExpr); !ok {
		t.Fatalf("应为检查表达式，实际 %T", decl.Value)
	}
}

func TestTryCatch(t *testing.T) {
	src := "尝试\n    打印(1)\n捕获 e\n    打印(e)\n结束"
	prog := mustParse(t, src)
	try, ok := prog.Statements[0].(*ast.TryStmt)
	if !ok {
		t.Fatalf("应为尝试语句，实际 %T", prog.Statements[0])
	}
	if try.CatchName != "e" || try.Catch == nil {
		t.Fatalf("捕获解析错误：%+v", try)
	}
}

func TestErrorType(t *testing.T) {
	src := "函数 f() -> 错误\n    返回 空\n结束"
	prog := mustParse(t, src)
	fn := prog.Statements[0].(*ast.VarDecl).Value.(*ast.FunctionLiteral)
	if len(fn.ReturnTypes) != 1 || fn.ReturnTypes[0] != ast.TypeError {
		t.Fatalf("错误返回类型解析错误：%+v", fn.ReturnTypes)
	}
}

func TestImportBasic(t *testing.T) {
	src := `导入 "mathlib/运算"`
	prog := mustParse(t, src)
	imp, ok := prog.Statements[0].(*ast.ImportStmt)
	if !ok {
		t.Fatalf("应为导入语句，实际 %T", prog.Statements[0])
	}
	if imp.Path != "mathlib/运算" || imp.Name != "运算" {
		t.Fatalf("导入解析错误：%+v", imp)
	}
}

func TestImportAlias(t *testing.T) {
	src := `导入 "mathlib/运算" 作为 m`
	prog := mustParse(t, src)
	imp := prog.Statements[0].(*ast.ImportStmt)
	if imp.Name != "m" {
		t.Fatalf("别名解析错误：%s", imp.Name)
	}
}

func TestImportMissingPath(t *testing.T) {
	expectError(t, `导入`)
}

func TestExportVar(t *testing.T) {
	src := `导出 变量 x = 1`
	prog := mustParse(t, src)
	exp, ok := prog.Statements[0].(*ast.ExportStmt)
	if !ok {
		t.Fatalf("应为导出语句，实际 %T", prog.Statements[0])
	}
	if exp.Name != "x" {
		t.Fatalf("导出名错误：%s", exp.Name)
	}
}

func TestExportFunction(t *testing.T) {
	src := "导出 函数 f()\n    返回 1\n结束"
	prog := mustParse(t, src)
	exp := prog.Statements[0].(*ast.ExportStmt)
	if exp.Name != "f" {
		t.Fatalf("导出函数名错误：%s", exp.Name)
	}
}

func TestExportClass(t *testing.T) {
	src := "导出 模型 点\n    变量 x\n结束"
	prog := mustParse(t, src)
	exp := prog.Statements[0].(*ast.ExportStmt)
	if exp.Name != "点" {
		t.Fatalf("导出模型名错误：%s", exp.Name)
	}
}
