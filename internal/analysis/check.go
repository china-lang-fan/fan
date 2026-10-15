package analysis

import (
	"fan/internal/ast"
)

// 软关键字：作为独立标识符出现时透明忽略，不参与运算，也不算未定义。
var softKeywords = map[string]bool{
	"通过": true,
	"从":  true,
}

// checkUndefinedNames 检查使用了但未在作用域链、顶层定义、模型、导入模块或
// 内建名字中声明的标识符。第一版仅做保守提示，避免误伤动态特性。
func (d *Document) checkUndefinedNames(idx *Index) []Diagnostic {
	if d.Prog == nil {
		return nil
	}
	defined := map[string]bool{}
	for name := range d.byName {
		defined[name] = true
	}
	for name := range d.classes {
		defined[name] = true
	}
	for name := range d.interfaces {
		defined[name] = true
	}
	for _, sym := range d.topLevels {
		if sym.fn != nil && len(sym.fn.NameSegments) >= 2 {
			defined[sym.fn.NameSegments[0]] = true
		}
	}
	for name := range d.imports {
		defined[name] = true
	}
	for name := range builtinNames {
		defined[name] = true
	}
	for name := range typeKeywords {
		defined[name] = true
	}
	// 函数内部还可能声明局部模型，先把所有类名加入全局可见集合。
	walk(d.Prog, func(node ast.Node) bool {
		if class, ok := node.(*ast.ClassStmt); ok {
			defined[class.Name] = true
		}
		if iface, ok := node.(*ast.InterfaceStmt); ok {
			defined[iface.Name] = true
			for _, embed := range iface.Embeds {
				defined[embed] = true
			}
		}
		return true
	})

	var diags []Diagnostic

	// 顶层语句：在全局名字集合下检查表达式中的标识符。
	diags = append(diags, d.checkTopLevelExpressions(defined)...)

	// 顶层语句各自模拟一次作用域。
	for _, stmt := range d.Prog.Statements {
		inner, _ := unwrapExport(stmt)
		if fnVar, ok := inner.(*ast.VarDecl); ok {
			if fn, ok := fnVar.Value.(*ast.FunctionLiteral); ok {
				diags = append(diags, checkFunctionBody(d, fn, defined)...)
				continue
			}
		}
		if method, ok := inner.(*ast.MethodDef); ok && method.Function != nil {
			diags = append(diags, checkFunctionBody(d, method.Function, defined)...)
		}
	}
	return diags
}

// checkTopLevelExpressions 检查顶层（函数体外）表达式中的未定义标识符。
func (d *Document) checkTopLevelExpressions(globals map[string]bool) []Diagnostic {
	var diags []Diagnostic
	for _, stmt := range d.Prog.Statements {
		inner, _ := unwrapExport(stmt)
		switch s := inner.(type) {
		case *ast.VarDecl:
			if _, isFn := s.Value.(*ast.FunctionLiteral); isFn {
				continue
			}
			if s.Value != nil {
				diags = append(diags, checkExprUndefined(s.Value, globals)...)
			}
		case *ast.ExpressionStmt:
			diags = append(diags, checkExprUndefined(s.Expression, globals)...)
		case *ast.AssignStmt:
			diags = append(diags, checkExprUndefined(s.Value, globals)...)
		case *ast.CompoundAssignStmt:
			diags = append(diags, checkExprUndefined(s.Target, globals)...)
			diags = append(diags, checkExprUndefined(s.Value, globals)...)
		case *ast.ReturnStmt:
			for _, v := range s.Values {
				diags = append(diags, checkExprUndefined(v, globals)...)
			}
		case *ast.ImportStmt:
			// 模块名已在 globals 中。
		}
	}
	return diags
}

func checkExprUndefined(expr ast.Expression, globals map[string]bool) []Diagnostic {
	var diags []Diagnostic
	walk(expr, func(node ast.Node) bool {
		// 不进入嵌套函数字面量，其内部由函数作用域检查。
		if _, ok := node.(*ast.FunctionLiteral); ok && node != expr {
			return false
		}
		ident, ok := node.(*ast.Identifier)
		if !ok {
			return true
		}
		if globals[ident.Name] || softKeywords[ident.Name] {
			return true
		}
		if _, ok := builtinNames[ident.Name]; ok {
			return true
		}
		diags = append(diags, Diagnostic{
			Line:     ident.Position.Line,
			Column:   ident.Position.Column,
			EndLine:  ident.Position.Line,
			EndCol:   ident.Position.Column + len([]rune(ident.Name)),
			Severity: SeverityWarning,
			Code:     "undefined-name",
			Message:  "未定义的名字：" + ident.Name,
		})
		return true
	})
	return diags
}

func checkFunctionBody(d *Document, fn *ast.FunctionLiteral, globals map[string]bool) []Diagnostic {
	localMap := collectLocalScopes(fn)
	var diags []Diagnostic
	for ident, locals := range localMap {
		name := ident.Name
		if globals[name] {
			continue
		}
		if _, ok := locals[name]; ok {
			continue
		}
		if softKeywords[name] {
			continue
		}
		if _, ok := builtinNames[name]; ok {
			continue
		}
		diags = append(diags, Diagnostic{
			Line:     ident.Position.Line,
			Column:   ident.Position.Column,
			EndLine:  ident.Position.Line,
			EndCol:   ident.Position.Column + len([]rune(name)),
			Severity: SeverityWarning,
			Code:     "undefined-name",
			Message:  "未定义的名字：" + name,
		})
	}
	return diags
}

// checkImports 检查导入路径能否静态解析。
func (d *Document) checkImports(idx *Index) []Diagnostic {
	var diags []Diagnostic
	for _, imp := range d.imports {
		if _, ok := idx.resolve(imp.path); ok {
			continue
		}
		diags = append(diags, Diagnostic{
			Line:     imp.namePos.Line,
			Column:   imp.namePos.Column,
			EndLine:  imp.namePos.Line,
			EndCol:   imp.namePos.Column + len([]rune(imp.path)) + 2,
			Severity: SeverityError,
			Code:     "import-not-found",
			Message:  "找不到模块：" + imp.path,
		})
	}
	return diags
}
