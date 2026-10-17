package analysis

import (
	"strings"

	"fan/internal/ast"
	"fan/internal/parser"
)

// Document 是单个凡语言源文件的静态分析视图。
type Document struct {
	URI    string
	Source string
	lines  []string
	Prog   *ast.Program

	// topLevels 按声明出现顺序保存顶层符号。
	topLevels []*Symbol
	// byName 便于按名字查找顶层符号（函数可重载，保留全部）。
	byName map[string][]*Symbol
	// classes 按模型名保存模型符号。
	classes map[string]*Symbol
	// interfaces 按接口名保存接口符号。
	interfaces map[string]*Symbol
	// namedTypes 按自定义类型名保存类型符号。
	namedTypes map[string]*Symbol
	// imports 按 导入时绑定的模块名 保存导入信息。
	imports map[string]*importInfo
	// fnRanges 记录函数体的行范围，供按光标位置找所在函数。
	fnRanges []fnRange
}

type importInfo struct {
	path    string
	namePos ast.Position
}

// SymbolKind 标识符号种类。
type SymbolKind int

const (
	KindFunction SymbolKind = iota
	KindMixfixFunction
	KindClass
	KindInterface
	KindType
	KindField
	KindMethod
	KindVariable
	KindConstant
	KindParameter
	KindModule
)

// Symbol 是静态收集到的一个符号。
type Symbol struct {
	Name     string
	Kind     SymbolKind
	Detail   string // 类型或签名
	Doc      string
	Pos      ast.Position
	EndPos   ast.Position
	Owner    string // 字段/方法所属模型，或模块名
	Children []*Symbol

	// 函数符号专用
	fn *ast.FunctionLiteral
}

// ParseDocument 解析并建立单文件符号视图。解析出错时仍返回
// 已能构建的部分程序，供诊断使用。
func ParseDocument(uri, source string) *Document {
	prog, _ := parser.ParseProgram(source)
	doc := &Document{
		URI:        uri,
		Source:     source,
		lines:      strings.Split(source, "\n"),
		Prog:       prog,
		byName:     map[string][]*Symbol{},
		classes:    map[string]*Symbol{},
		interfaces: map[string]*Symbol{},
		namedTypes: map[string]*Symbol{},
		imports:    map[string]*importInfo{},
	}
	if prog != nil {
		doc.collect()
		doc.collectFunctionRanges()
	}
	return doc
}

// fnRange 记录一个函数体在文件中的行范围（1-based）。
type fnRange struct {
	fn        *ast.FunctionLiteral
	startLine int
	endLine   int
}

func (d *Document) collectFunctionRanges() {
	walk(d.Prog, func(node ast.Node) bool {
		fn, ok := node.(*ast.FunctionLiteral)
		if !ok || fn.Body == nil {
			return true
		}
		end := fn.Position.Line
		for _, stmt := range fn.Body.Statements {
			if line := nodeMaxLine(stmt); line > end {
				end = line
			}
		}
		d.fnRanges = append(d.fnRanges, fnRange{fn: fn, startLine: fn.Position.Line, endLine: end})
		return true
	})
}

func nodeMaxLine(node ast.Node) int {
	maxLine := node.Pos().Line
	walk(node, func(n ast.Node) bool {
		if n.Pos().Line > maxLine {
			maxLine = n.Pos().Line
		}
		return true
	})
	return maxLine
}

// functionAtLine 返回包含指定 0-based 行的最内层函数。
func (d *Document) functionAtLine(line0 int) *ast.FunctionLiteral {
	line1 := line0 + 1
	var result *ast.FunctionLiteral
	bestSpan := 1 << 30
	for _, r := range d.fnRanges {
		if line1 >= r.startLine && line1 <= r.endLine {
			span := r.endLine - r.startLine
			if span < bestSpan {
				bestSpan = span
				result = r.fn
			}
		}
	}
	return result
}

func (d *Document) collect() {
	for _, stmt := range d.Prog.Statements {
		d.collectStatement(stmt, "")
	}
}

func (d *Document) add(sym *Symbol) {
	d.topLevels = append(d.topLevels, sym)
	d.byName[sym.Name] = append(d.byName[sym.Name], sym)
}

func unwrapVisibility(stmt ast.Statement) (ast.Statement, bool) {
	if export, ok := stmt.(*ast.VisibilityStmt); ok {
		return export.Inner, true
	}
	return stmt, false
}

func (d *Document) collectStatement(stmt ast.Statement, owner string) {
	inner, _ := unwrapVisibility(stmt)
	switch s := inner.(type) {
	case *ast.VarDecl:
		if fn, ok := s.Value.(*ast.FunctionLiteral); ok {
			d.collectFunction(s.Name, fn, s.Position)
			return
		}
		kind := KindVariable
		if s.IsConst {
			kind = KindConstant
		}
		sym := &Symbol{
			Name: s.Name, Kind: kind, Pos: s.Position,
			EndPos: endOf(d.lines, s.Position),
			Detail: string(s.DeclType),
			Doc:    d.docBefore(s.Position.Line),
		}
		d.add(sym)
	case *ast.ClassStmt:
		classSym := &Symbol{
			Name: s.Name, Kind: KindClass, Pos: s.Position,
			EndPos: endOf(d.lines, s.Position),
			Doc:    d.docBefore(s.Position.Line),
		}
		for _, field := range s.Fields {
			classSym.Children = append(classSym.Children, &Symbol{
				Name: field.Name, Kind: KindField, Owner: s.Name,
				Detail: string(field.Type),
				Pos:    field.Position,
				EndPos: ast.Position{Line: field.Position.Line, Column: field.Position.Column + len([]rune(field.Name))},
				Doc:    d.docBefore(field.Position.Line),
			})
		}
		d.add(classSym)
		d.classes[s.Name] = classSym
	case *ast.InterfaceStmt:
		ifaceSym := &Symbol{
			Name: s.Name, Kind: KindInterface, Pos: s.Position,
			EndPos: endOf(d.lines, s.Position),
			Doc:    d.docBefore(s.Position.Line),
		}
		for _, m := range s.Methods {
			ifaceSym.Children = append(ifaceSym.Children, &Symbol{
				Name: m.Name, Kind: KindMethod, Owner: s.Name,
				Detail: interfaceMethodSignature(m),
				Pos:    m.Position,
				EndPos: ast.Position{Line: m.Position.Line, Column: m.Position.Column + len([]rune(m.Name))},
			})
		}
		d.add(ifaceSym)
		d.interfaces[s.Name] = ifaceSym
	case *ast.TypeDefStmt:
		sym := &Symbol{
			Name: s.Name, Kind: KindType, Pos: s.Position,
			EndPos: endOf(d.lines, s.Position),
			Detail: string(s.BaseType),
			Doc:    d.docBefore(s.Position.Line),
		}
		d.add(sym)
		d.namedTypes[s.Name] = sym
	case *ast.MethodDef:
		sym := &Symbol{
			Name: s.MethodName, Kind: KindMethod, Owner: s.ClassName,
			Pos: s.Position, EndPos: endOf(d.lines, s.Position),
			Detail: methodSignature(s),
			Doc:    d.docBefore(s.Position.Line),
			fn:     s.Function,
		}
		d.add(sym)
		if class, ok := d.classes[s.ClassName]; ok {
			class.Children = append(class.Children, sym)
		}
	case *ast.ImportStmt:
		d.imports[s.Name] = &importInfo{path: s.Path, namePos: s.Position}
	}
}

func methodSignature(s *ast.MethodDef) string {
	if s.Function == nil {
		return s.MethodName + "()"
	}
	fn := *s.Function
	fn.Name = s.MethodName
	return simpleSignature(&fn)
}

func interfaceMethodSignature(m *ast.InterfaceMethodSig) string {
	var builder strings.Builder
	builder.WriteString(m.Name)
	builder.WriteString("(")
	for i, param := range m.Params {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(param.Name)
		if param.Type != "" && param.Type != ast.TypeAny {
			builder.WriteString(" ")
			builder.WriteString(string(param.Type))
		}
	}
	builder.WriteString(")")
	if len(m.ReturnTypes) > 0 {
		builder.WriteString(" -> ")
		for i, dt := range m.ReturnTypes {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(string(dt))
		}
	}
	return builder.String()
}

func (d *Document) collectFunction(name string, fn *ast.FunctionLiteral, pos ast.Position) {
	kind := KindFunction
	detail := simpleSignature(fn)
	if len(fn.NameSegments) >= 2 {
		kind = KindMixfixFunction
		detail = fn.Signature
	}
	sym := &Symbol{
		Name: name, Kind: kind, Pos: pos,
		EndPos: endOf(d.lines, pos),
		Detail: detail,
		Doc:    d.docBefore(pos.Line),
		fn:     fn,
	}
	if fn.Body != nil {
		for _, param := range fn.Params {
			sym.Children = append(sym.Children, &Symbol{
				Name: param.Name, Kind: KindParameter,
				Detail: string(param.Type), Owner: name,
			})
		}
	}
	d.add(sym)
}

func simpleSignature(fn *ast.FunctionLiteral) string {
	var builder strings.Builder
	builder.WriteString(fn.Name)
	builder.WriteString("(")
	for i, param := range fn.Params {
		if i > 0 {
			builder.WriteString(", ")
		}
		if param.Variadic {
			builder.WriteString("...")
		}
		builder.WriteString(param.Name)
		if param.Type != ast.TypeAny {
			builder.WriteString(" ")
			builder.WriteString(string(param.Type))
		}
	}
	builder.WriteString(")")
	if len(fn.ReturnTypes) > 0 {
		builder.WriteString(" -> ")
		for i, t := range fn.ReturnTypes {
			if i > 0 {
				builder.WriteString("、")
			}
			builder.WriteString(string(t))
		}
	}
	return builder.String()
}

func endOf(lines []string, pos ast.Position) ast.Position {
	if pos.Line-1 < 0 || pos.Line-1 >= len(lines) {
		return pos
	}
	return ast.Position{Line: pos.Line, Column: len([]rune(lines[pos.Line-1])) + 1}
}

// docBefore 提取紧贴指定行（1-based）上方、中间没有空行的连续注释块。
// 行内尾随注释不算文档注释。支持 “# ...” 与 “注释 ...” 两种写法。
func (d *Document) docBefore(declLine int) string {
	var docs []string
	for line := declLine - 1; line >= 1; line-- {
		raw := d.lineText(line)
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			break
		}
		text, ok := docCommentText(trimmed)
		if !ok {
			break
		}
		docs = append([]string{text}, docs...)
	}
	return strings.Join(docs, "\n")
}

func (d *Document) lineText(line int) string {
	index := line - 1
	if index < 0 || index >= len(d.lines) {
		return ""
	}
	return d.lines[index]
}

// docCommentText 判断一行是否为整行注释，并返回去掉注释前缀的文本。
func docCommentText(trimmed string) (string, bool) {
	if strings.HasPrefix(trimmed, "#") {
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "#")), true
	}
	if strings.HasPrefix(trimmed, "注释") {
		rest := strings.TrimPrefix(trimmed, "注释")
		if rest == "" {
			return "", true
		}
		r := []rune(rest)[0]
		if r == ' ' || r == '\t' || r == ':' || r == '：' {
			return strings.TrimSpace(strings.TrimLeft(rest, " \t:：")), true
		}
	}
	return "", false
}
