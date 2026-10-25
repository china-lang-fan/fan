package analysis

import (
	"fan/internal/ast"
)

// inferType 在不执行代码的前提下推断表达式的静态类型名。
// 返回模型名、原生类型名（整数/小数/字符串/布尔/数组/字典）或空串（无法推断）。
func (idx *Index) inferType(expr ast.Expression, locals map[string]string) string {
	switch n := expr.(type) {
	case *ast.IntegerLiteral:
		return "整数"
	case *ast.FloatLiteral:
		return "小数"
	case *ast.StringLiteral:
		return "字符串"
	case *ast.InterpString:
		return "字符串"
	case *ast.CharLiteral:
		return "字符"
	case *ast.BoolLiteral:
		return "布尔"
	case *ast.ArrayLiteral:
		return "数组"
	case *ast.DictLiteral:
		return "字典"
	case *ast.SliceExpr:
		return idx.inferType(n.Target, locals)
	case *ast.SpreadExpr:
		return idx.inferType(n.Value, locals)
	case *ast.CompositeLiteral:
		if id, ok := n.Model.(*ast.Identifier); ok {
			return id.Name
		}
		if m, ok := n.Model.(*ast.MemberExpr); ok {
			return m.Name
		}
		return ""
	case *ast.Identifier:
		return idx.inferIdentifier(n.Name, locals)
	case *ast.CallExpr:
		return idx.inferCallType(n, locals)
	case *ast.MaybeCallExpr:
		return idx.inferType(n.Callee, locals)
	case *ast.MemberExpr:
		return idx.inferMemberType(n, locals)
	case *ast.ImplicitReceiverCallExpr:
		// 直接连接分段函数，静态返回类型难以推断。
		return ""
	case *ast.TernaryExpr:
		t := idx.inferType(n.Then, locals)
		if t != "" {
			return t
		}
		return idx.inferType(n.Else, locals)
	case *ast.TypeTestExpr:
		return "布尔"
	case *ast.TypeCastExpr:
		return n.TypeName
	}
	return ""
}

func (idx *Index) inferIdentifier(name string, locals map[string]string) string {
	if locals != nil {
		if t, ok := locals[name]; ok && t != "" {
			return t
		}
	}
	// 局部/顶层变量声明类型
	if syms := idx.root.byName[name]; len(syms) > 0 {
		sym := syms[len(syms)-1]
		if sym.Detail != "" {
			return sym.Detail
		}
		if sym.fn != nil && len(sym.fn.ReturnTypes) > 0 {
			return string(sym.fn.ReturnTypes[0])
		}
	}
	return ""
}

func (idx *Index) inferCallType(call *ast.CallExpr, locals map[string]string) string {
	// 模型构造：标识符是模型名
	if ident, ok := call.Callee.(*ast.Identifier); ok {
		if _, ok := idx.root.classes[ident.Name]; ok {
			return ident.Name
		}
		// 自定义类型转换：状态(0) 推断为 状态
		if _, ok := idx.root.namedTypes[ident.Name]; ok {
			return ident.Name
		}
		// 普通函数返回类型
		if syms := idx.root.byName[ident.Name]; len(syms) > 0 {
			sym := syms[len(syms)-1]
			if sym.fn != nil && len(sym.fn.ReturnTypes) > 0 {
				return string(sym.fn.ReturnTypes[0])
			}
		}
		// 跨模块调用：模块.函数
		if member, ok := call.Callee.(*ast.MemberExpr); ok {
			_ = member
		}
	}
	if member, ok := call.Callee.(*ast.MemberExpr); ok {
		if fn := idx.findMemberFunction(member, locals); fn != nil && len(fn.fn.ReturnTypes) > 0 {
			return string(fn.fn.ReturnTypes[0])
		}
	}
	return ""
}

func (idx *Index) inferMemberType(member *ast.MemberExpr, locals map[string]string) string {
	ownerType := idx.expressionOwnerType(member.Object, locals)
	if ownerType == "" {
		return ""
	}
	// 模型字段类型
	if class := idx.root.classes[ownerType]; class != nil {
		for _, child := range class.Children {
			if child.Name == member.Name {
				return child.Detail
			}
		}
		return ""
	}
	// 原生类型方法返回类型
	if fn := idx.lookupMethod(ownerType, member.Name); fn != nil && len(fn.fn.ReturnTypes) > 0 {
		return string(fn.fn.ReturnTypes[0])
	}
	return ""
}

// expressionOwnerType 推断成员表达式左半部分的类型（模型名或原生类型名）。
func (idx *Index) expressionOwnerType(expr ast.Expression, locals map[string]string) string {
	switch n := expr.(type) {
	case *ast.Identifier:
		return idx.inferIdentifier(n.Name, locals)
	case *ast.CallExpr:
		return idx.inferCallType(n, locals)
	case *ast.MemberExpr:
		return idx.inferMemberType(n, locals)
	case *ast.MaybeCallExpr:
		return idx.inferType(n.Callee, locals)
	}
	return ""
}

func (idx *Index) findMemberFunction(member *ast.MemberExpr, locals map[string]string) *Symbol {
	owner := idx.expressionOwnerType(member.Object, locals)
	if owner == "" {
		return nil
	}
	return idx.lookupMethod(owner, member.Name)
}

func (idx *Index) lookupMethod(ownerType, methodName string) *Symbol {
	// 模型方法
	for _, sym := range idx.root.byName[methodName] {
		if sym.Kind == KindMethod && sym.Owner == ownerType {
			return sym
		}
	}
	// 跨模块方法（定义 字符串 的 方法 … 定义在 SDK 文件）
	for _, doc := range idx.docs {
		for _, sym := range doc.byName[methodName] {
			if sym.Kind == KindMethod && sym.Owner == ownerType {
				return sym
			}
		}
	}
	// 入口文件内定义的原生类型方法
	for _, sym := range idx.root.byName[methodName] {
		if sym.Kind == KindMethod && sym.Owner == ownerType {
			return sym
		}
	}
	return nil
}

// localTypesForFunction 计算函数体内局部变量名到推断类型的映射。
func (idx *Index) localTypesForFunction(fn *ast.FunctionLiteral) map[string]string {
	locals := map[string]string{}
	if fn == nil || fn.Body == nil {
		return locals
	}
	for _, param := range fn.Params {
		if param.Type != ast.TypeAny {
			locals[param.Name] = string(param.Type)
		}
	}
	var visit func(node ast.Node)
	visit = func(node ast.Node) {
		if node == nil {
			return
		}
		switch n := node.(type) {
		case *ast.VarDecl:
			t := string(n.DeclType)
			if t == "" || t == "任意" {
				t = idx.inferType(n.Value, locals)
			}
			locals[n.Name] = t
		case *ast.MultiDecl:
			if cast, ok := n.Value.(*ast.TypeCastExpr); ok && len(n.Names) == 2 {
				locals[n.Names[0]] = cast.TypeName
				locals[n.Names[1]] = "布尔"
				break
			}
			if tuple, ok := n.Value.(*ast.TupleExpr); ok && len(tuple.Elements) == len(n.Names) {
				for i, name := range n.Names {
					if t := idx.inferType(tuple.Elements[i], locals); t != "" {
						locals[name] = t
					}
				}
				break
			}
			if t := idx.inferType(n.Value, locals); t != "" {
				for _, name := range n.Names {
					locals[name] = t
				}
			}
		}
		for _, child := range childrenOf(node) {
			visit(child)
		}
	}
	for _, stmt := range fn.Body.Statements {
		visit(stmt)
	}
	return locals
}

// memberOwnerAtPosition 保留供调用方传入局部类型映射使用。
func (idx *Index) memberOwnerAtPosition(line0 int, ownerExpr string, fn *ast.FunctionLiteral, locals map[string]string) string {
	if locals != nil {
		if t, ok := locals[ownerExpr]; ok && t != "" {
			return t
		}
	}
	return idx.inferIdentifier(ownerExpr, locals)
}
