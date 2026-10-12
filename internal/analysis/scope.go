package analysis

import (
	"fan/internal/ast"
)

// scopeSymbol 是函数内部静态可见的一个局部名字。
type scopeSymbol struct {
	name string
	pos  ast.Position
	typ  ast.DeclType
	kind SymbolKind
}

// collectLocalScopes 计算函数体内各标识符节点可见的局部/参数名字。
// 采用“先进入新作用域 → 手工处理会声明名字的语句 → 其余子节点通用遍历”
// 的方式，保证变量声明后才可见（不提升）。
func collectLocalScopes(fn *ast.FunctionLiteral) map[*ast.Identifier]map[string]scopeSymbol {
	result := map[*ast.Identifier]map[string]scopeSymbol{}
	if fn == nil || fn.Body == nil {
		return result
	}
	var scopes []map[string]scopeSymbol
	push := func() { scopes = append(scopes, map[string]scopeSymbol{}) }
	pop := func() { scopes = scopes[:len(scopes)-1] }
	declare := func(name string, typ ast.DeclType, kind SymbolKind, pos ast.Position) {
		scopes[len(scopes)-1][name] = scopeSymbol{name: name, pos: pos, typ: typ, kind: kind}
	}
	visible := func() map[string]scopeSymbol {
		out := map[string]scopeSymbol{}
		for _, scope := range scopes {
			for name, sym := range scope {
				out[name] = sym
			}
		}
		return out
	}

	push()
	for _, param := range fn.Params {
		declare(param.Name, param.Type, KindParameter, fn.Position)
	}
	declare("自己", ast.TypeAny, KindVariable, fn.Position)

	var visit func(node ast.Node)
	visitBlock := func(block *ast.BlockStmt) {
		if block == nil {
			return
		}
		push()
		for _, stmt := range block.Statements {
			visit(stmt)
		}
		pop()
	}
	visit = func(node ast.Node) {
		if node == nil {
			return
		}
		switch n := node.(type) {
		case *ast.FunctionLiteral:
			if n != fn {
				// 嵌套函数参数/局部对外不可见，但其内部标识符同样记录可见名字。
				nested := collectLocalScopes(n)
				for ident, names := range nested {
					if _, exists := result[ident]; !exists {
						result[ident] = names
					}
				}
				return
			}
		case *ast.BlockStmt:
			visitBlock(n)
			return
		case *ast.VarDecl:
			if n.Value != nil {
				visit(n.Value)
			}
			kind := KindVariable
			if n.IsConst {
				kind = KindConstant
			}
			declare(n.Name, n.DeclType, kind, n.Position)
			return
		case *ast.MultiDecl:
			if n.Value != nil {
				visit(n.Value)
			}
			for _, name := range n.Names {
				declare(name, ast.TypeAny, KindVariable, n.Position)
			}
			return
		case *ast.ForEachStmt:
			if n.Iterable != nil {
				visit(n.Iterable)
			}
			push()
			declare(n.FirstName, ast.TypeAny, KindVariable, n.Position)
			if n.SecondName != "" {
				declare(n.SecondName, ast.TypeAny, KindVariable, n.Position)
			}
			visitBlock(n.Body)
			pop()
			return
		case *ast.Identifier:
			result[n] = visible()
		}
		for _, child := range childrenOf(node) {
			visit(child)
		}
	}

	for _, stmt := range fn.Body.Statements {
		visit(stmt)
	}
	pop()
	return result
}
