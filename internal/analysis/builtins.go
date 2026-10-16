package analysis

import "fan/internal/ast"

// builtinNames 是 Go 内建函数与常见全局名字的集合，用于未定义诊断排除。
// 不提供悬停说明（内建函数没有源码注释）。
var builtinNames = map[string]bool{}

// typeKeywords 是类型关键字，出现在表达式位置时不作为未定义名字。
var typeKeywords = map[string]bool{}

func init() {
	for _, name := range []string{
		"长度", "追加", "打印", "错误", "注册标签处理", "捕获",
		"type", "len", "append", "error",
	} {
		builtinNames[name] = true
	}
	for _, t := range []ast.DeclType{
		ast.TypeInt, ast.TypeFloat, ast.TypeString, ast.TypeBool,
		ast.TypeArray, ast.TypeDict, ast.TypeError, ast.TypeByte,
		ast.TypeChar, ast.TypeAny,
	} {
		typeKeywords[string(t)] = true
	}
	typeKeywords["任意"] = true
}
