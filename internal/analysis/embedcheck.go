package analysis

import (
	"fan/internal/ast"
)

// classStaticInfo 是嵌入歧义检查用的静态模型信息。
type classStaticInfo struct {
	embeds  []string
	fields  map[string]bool
	methods map[string]bool
}

// checkEmbedAmbiguity 对类型可静态确定的成员访问做同层嵌入歧义诊断：
// 同层多个嵌入提供同名字段或方法时提示用嵌入名限定。
// 仅在类型信息明确时报告，避免对鸭子类型写法误伤。
func (idx *Index) checkEmbedAmbiguity() []Diagnostic {
	infos := idx.buildClassInfo()
	var diags []Diagnostic
	report := func(pos ast.Position, owner, name string) {
		if owner == "" {
			return
		}
		info := infos[owner]
		if info == nil {
			return
		}
		if ambiguousField(infos, info, name, map[string]bool{}) ||
			ambiguousMethod(infos, info, name, map[string]bool{}) {
			diags = append(diags, Diagnostic{
				Line:     pos.Line,
				Column:   pos.Column,
				EndLine:  pos.Line,
				EndCol:   pos.Column + len([]rune(owner)),
				Severity: SeverityError,
				Code:     "embed-ambiguous",
				Message:  owner + " 的成员 " + name + " 在同层嵌入中有歧义，应用 嵌入名." + name + " 限定",
			})
		}
	}
	walk(idx.root.Prog, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.MemberExpr:
			report(n.Position, idx.staticOwnerType(n.Object), n.Name)
		case *ast.FieldAssignExpr:
			report(n.Position, idx.staticOwnerType(n.Object), n.Name)
		}
		return true
	})
	return diags
}

// staticOwnerType 仅对“类型明确”的接收者表达式返回模型名：
// 构造调用 模型(...)、具名类型变量、嵌入链成员。其余返回空串。
func (idx *Index) staticOwnerType(expr ast.Expression) string {
	switch n := expr.(type) {
	case *ast.CallExpr:
		if ident, ok := n.Callee.(*ast.Identifier); ok {
			if idx.lookupClassAnyDoc(ident.Name) != nil {
				return ident.Name
			}
		}
	case *ast.Identifier:
		return idx.typedIdentifier(n.Name)
	case *ast.MemberExpr:
		// d.动物 的类型是被嵌入模型。
		owner := idx.staticOwnerType(n.Object)
		if owner == "" {
			return ""
		}
		if class := idx.lookupClassAnyDoc(owner); class != nil {
			for _, embedName := range allEmbeds(class, idx, map[string]bool{}) {
				if embedName == n.Name {
					return n.Name
				}
			}
		}
	}
	return ""
}

// typedIdentifier 返回文件内可静态确定的标识符类型：
// 带类型标注的参数、变量声明，或带类型的调用结果变量（保守处理）。
func (idx *Index) typedIdentifier(name string) string {
	// 顶层/模块中带类型声明的变量
	for _, doc := range idx.allDocs() {
		for _, sym := range doc.byName[name] {
			if (sym.Kind == KindVariable || sym.Kind == KindParameter) && sym.Detail != "" {
				if idx.lookupClassAnyDoc(sym.Detail) != nil {
					return sym.Detail
				}
			}
		}
	}
	// 函数参数与局部变量（仅在包含该名字的函数作用域内判定）
	var found string
	walk(idx.root.Prog, func(node ast.Node) bool {
		if found != "" {
			return false
		}
		fn, ok := node.(*ast.FunctionLiteral)
		if !ok {
			return true
		}
		for _, p := range fn.Params {
			if p.Name == name && p.Type != ast.TypeAny && idx.lookupClassAnyDoc(string(p.Type)) != nil {
				found = string(p.Type)
				return false
			}
		}
		for _, local := range collectTypedLocals(fn) {
			if local.name == name && idx.lookupClassAnyDoc(local.typ) != nil {
				found = local.typ
				return false
			}
		}
		return true
	})
	return found
}

type typedLocal struct {
	name string
	typ  string
}

// collectTypedLocals 收集函数体内带显式类型标注或模型构造初始化的局部变量。
func collectTypedLocals(fn *ast.FunctionLiteral) []typedLocal {
	var out []typedLocal
	if fn.Body == nil {
		return out
	}
	for _, stmt := range fn.Body.Statements {
		collectTypedLocalsStmt(stmt, &out)
	}
	return out
}

func collectTypedLocalsStmt(stmt ast.Statement, out *[]typedLocal) {
	switch s := stmt.(type) {
	case *ast.VarDecl:
		if s.DeclType != ast.TypeAny {
			*out = append(*out, typedLocal{name: s.Name, typ: string(s.DeclType)})
			return
		}
		if call, ok := s.Value.(*ast.CallExpr); ok {
			if ident, ok := call.Callee.(*ast.Identifier); ok {
				*out = append(*out, typedLocal{name: s.Name, typ: ident.Name})
			}
		}
	case *ast.MultiDecl:
		// 多声明不支持单类型标注，跳过。
	default:
		walk(stmt, func(node ast.Node) bool {
			if _, ok := node.(*ast.FunctionLiteral); ok {
				return false
			}
			decl, ok := node.(*ast.VarDecl)
			if !ok {
				return true
			}
			if decl.DeclType != ast.TypeAny {
				*out = append(*out, typedLocal{name: decl.Name, typ: string(decl.DeclType)})
				return true
			}
			if call, ok := decl.Value.(*ast.CallExpr); ok {
				if ident, ok := call.Callee.(*ast.Identifier); ok {
					*out = append(*out, typedLocal{name: decl.Name, typ: ident.Name})
				}
			}
			return true
		})
	}
}

func (idx *Index) buildClassInfo() map[string]*classStaticInfo {
	infos := map[string]*classStaticInfo{}
	for _, doc := range idx.allDocs() {
		for name, class := range doc.classes {
			info := &classStaticInfo{
				embeds:  append([]string(nil), class.Embeds...),
				fields:  map[string]bool{},
				methods: map[string]bool{},
			}
			for _, child := range class.Children {
				switch child.Kind {
				case KindField:
					info.fields[child.Name] = true
				case KindMethod:
					info.methods[child.Name] = true
				}
			}
			infos[name] = info
		}
		// 方法定义在模型语句之外。
		for _, syms := range doc.byName {
			for _, sym := range syms {
				if sym.Kind == KindMethod && sym.Owner != "" {
					info := infos[sym.Owner]
					if info == nil {
						info = &classStaticInfo{fields: map[string]bool{}, methods: map[string]bool{}}
						infos[sym.Owner] = info
					}
					info.methods[sym.Name] = true
				}
			}
		}
	}
	return infos
}

func ambiguousField(infos map[string]*classStaticInfo, info *classStaticInfo, name string, seen map[string]bool) bool {
	if info.fields[name] {
		return false
	}
	count := 0
	for _, embedName := range info.embeds {
		if seen[embedName] {
			continue
		}
		seen[embedName] = true
		embed := infos[embedName]
		if embed == nil {
			continue
		}
		if memberExists(infos, embed, name, true) {
			count++
		}
	}
	return count >= 2
}

func ambiguousMethod(infos map[string]*classStaticInfo, info *classStaticInfo, name string, seen map[string]bool) bool {
	if info.methods[name] {
		return false
	}
	count := 0
	for _, embedName := range info.embeds {
		if seen[embedName] {
			continue
		}
		seen[embedName] = true
		embed := infos[embedName]
		if embed == nil {
			continue
		}
		if memberExists(infos, embed, name, false) {
			count++
		}
	}
	return count >= 2
}

func memberExists(infos map[string]*classStaticInfo, info *classStaticInfo, name string, field bool) bool {
	if field {
		if info.fields[name] {
			return true
		}
	} else if info.methods[name] {
		return true
	}
	for _, embedName := range info.embeds {
		if embed := infos[embedName]; embed != nil {
			if memberExists(infos, embed, name, field) {
				return true
			}
		}
	}
	return false
}

func allEmbeds(class *Symbol, idx *Index, seen map[string]bool) []string {
	var out []string
	for _, name := range class.Embeds {
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
		if embed := idx.lookupClassAnyDoc(name); embed != nil {
			out = append(out, allEmbeds(embed, idx, seen)...)
		}
	}
	return out
}
