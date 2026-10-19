// Package analysis 在不执行源码的前提下对凡语言程序做静态分析，
// 为 LSP、lint 等工具提供语法错误与风格诊断。
package analysis

import (
	"strings"
	"unicode/utf16"

	"fan/internal/ast"
	"fan/internal/parser"
)

// Severity 表示诊断严重级别。
type Severity int

const (
	SeverityError Severity = iota + 1
	SeverityWarning
	SeverityInformation
	SeverityHint
)

// Diagnostic 是与具体编辑器协议无关的诊断结构。
// 行列均为 1-based，列以 Unicode 码点（rune）计。
type Diagnostic struct {
	Line     int      `json:"line"`
	Column   int      `json:"column"`
	EndLine  int      `json:"endLine"`
	EndCol   int      `json:"endColumn"`
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
}

// Analyze 对一份源码做静态分析。
func Analyze(source string) []Diagnostic {
	index := BuildIndex("", source)
	return AnalyzeIndex(index)
}

// AnalyzeIndex 基于跨文件索引做完整诊断，供 LSP 使用（含真实文件 URI）。
func AnalyzeIndex(index *Index) []Diagnostic {
	source := index.root.Source
	prog, errs := parser.ParseProgram(source)
	var diags []Diagnostic
	for _, e := range errs {
		diags = append(diags, Diagnostic{
			Line:     e.Line,
			Column:   e.Column,
			EndLine:  e.Line,
			EndCol:   e.Column + 1,
			Severity: SeverityError,
			Code:     "syntax",
			Message:  e.Message,
		})
	}
	if prog == nil {
		return diags
	}
	registry := collectMixfixDefs(prog)
	diags = append(diags, checkMixfixSpacing(prog, registry)...)
	diags = append(diags, index.root.checkImports(index)...)
	diags = append(diags, index.root.checkUndefinedNames(index)...)
	diags = append(diags, index.checkEmbedAmbiguity()...)
	diags = append(diags, index.checkCompositeLiterals()...)
	return diags
}

type mixfixDef struct {
	segments   []string
	connectors []string
	groupSizes []int
}

type mixfixRegistry struct {
	// roots 以首段名为键。
	roots map[string][]mixfixDef
}

func (r *mixfixRegistry) hasRoot(name string) bool {
	_, ok := r.roots[name]
	return ok
}

func collectMixfixDefs(prog *ast.Program) *mixfixRegistry {
	registry := &mixfixRegistry{roots: map[string][]mixfixDef{}}
	for _, stmt := range prog.Statements {
		fn := functionLiteralOf(stmt)
		if fn == nil || len(fn.NameSegments) < 2 {
			continue
		}
		root := fn.NameSegments[0]
		registry.roots[root] = append(registry.roots[root], mixfixDef{
			segments:   fn.NameSegments,
			connectors: fn.Connectors,
			groupSizes: fn.GroupSizes,
		})
	}
	return registry
}

func functionLiteralOf(stmt ast.Statement) *ast.FunctionLiteral {
	switch s := stmt.(type) {
	case *ast.VarDecl:
		if fn, ok := s.Value.(*ast.FunctionLiteral); ok {
			return fn
		}
	case *ast.VisibilityStmt:
		return functionLiteralOf(s.Inner)
	}
	return nil
}

// mixfixChain 与 evaluator 中的调用链结构对应，但只用于静态检查。
type mixfixChain struct {
	root         string
	segments     []string
	connectors   []string
	groupSizes   []int
	spaced       []bool
	rootPos      ast.Position
	connectorPos []ast.Position
}

// checkMixfixSpacing 检查分段函数调用的段间空白与签名匹配。
// 借助 walkWithParents 提供的直接父节点，只对最外层链诊断，避免重复。
func checkMixfixSpacing(prog *ast.Program, registry *mixfixRegistry) []Diagnostic {
	var diags []Diagnostic
	walkWithParents(prog, nil, func(node ast.Node, parent ast.Node) bool {
		expr, ok := node.(ast.Expression)
		if !ok {
			return true
		}
		chain, ok := collectChain(expr)
		if !ok || !registry.hasRoot(chain.root) {
			return true
		}
		if pexpr, ok := parent.(ast.Expression); ok {
			if pchain, ok := collectChain(pexpr); ok && pchain.root == chain.root &&
				len(pchain.segments) > len(chain.segments) {
				return true
			}
		}
		diags = append(diags, diagnoseChain(chain, registry)...)
		return true
	})
	return diags
}

func diagnoseChain(chain *mixfixChain, registry *mixfixRegistry) []Diagnostic {
	for i, spaced := range chain.spaced {
		if !spaced || i == 0 {
			continue
		}
		pos := chain.connectorPos[i]
		connector := chain.connectors[i]
		if connector == "" {
			connector = "的"
		}
		return []Diagnostic{{
			Line: pos.Line, Column: pos.Column,
			EndLine:  pos.Line,
			EndCol:   pos.Column + lenRunes(connector),
			Severity: SeverityWarning,
			Code:     "mixfix-space",
			Message: "分段函数各段之间不能有空格；这里会先普通调用 " + chain.root +
				"(...)，再访问返回值的成员，与分段函数是两件事。若要调用分段函数，请删除 “" + connector +
				"” 前的空格。",
		}}
	}
	if !matchesAnyDef(chain, registry) {
		pos := chain.rootPos
		return []Diagnostic{{
			Line: pos.Line, Column: pos.Column,
			EndLine:  pos.Line,
			EndCol:   pos.Column + lenRunes(chain.root),
			Severity: SeverityWarning,
			Code:     "mixfix-unresolved",
			Message: "此处按分段函数调用解析，但找不到名称与段数匹配的分段函数 " +
				chainSignature(chain) + "；请检查各段名称。",
		}}
	}
	return nil
}

func matchesAnyDef(chain *mixfixChain, registry *mixfixRegistry) bool {
	for _, def := range registry.roots[chain.root] {
		if len(def.segments) != len(chain.segments) {
			continue
		}
		ok := true
		for i := 1; i < len(def.segments); i++ {
			if def.segments[i] != chain.segments[i] || def.connectors[i] != chain.connectors[i] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func chainSignature(chain *mixfixChain) string {
	var builder strings.Builder
	for i, segment := range chain.segments {
		builder.WriteString(chain.connectors[i])
		builder.WriteString(segment)
		builder.WriteString("(...)")
	}
	return builder.String()
}

func lenRunes(s string) int { return len([]rune(s)) }

// collectChain 从调用链表达式中还原分段结构。
// 既接受显式括号调用，也接受 MaybeCallExpr 包裹的无括号末段。
func collectChain(node ast.Expression) (*mixfixChain, bool) {
	current := node
	type outSeg struct {
		connector    string
		name         string
		size         int
		spaced       bool
		connectorPos ast.Position
	}
	var outward []outSeg
	var rootArgs []ast.Expression
	for {
		switch n := current.(type) {
		case *ast.CallExpr:
			if ident, ok := n.Callee.(*ast.Identifier); ok {
				rootArgs = n.Args
				current = ident
				continue
			}
			member, ok := n.Callee.(*ast.MemberExpr)
			if !ok {
				return nil, false
			}
			outward = append(outward, outSeg{
				connector: member.Connector, name: member.Name,
				size: len(n.Args), spaced: member.Spaced, connectorPos: member.Position,
			})
			current = member.Object
		case *ast.ImplicitReceiverCallExpr:
			outward = append(outward, outSeg{
				connector: n.Connector, name: n.Name,
				size: len(n.Args), spaced: n.Spaced, connectorPos: n.Position,
			})
			current = n.Receiver
		case *ast.MemberExpr:
			outward = append(outward, outSeg{
				connector: n.Connector, name: n.Name,
				size: 0, spaced: n.Spaced, connectorPos: n.Position,
			})
			current = n.Object
		case *ast.MaybeCallExpr:
			current = n.Callee
		default:
			goto root
		}
	}
root:
	ident, ok := current.(*ast.Identifier)
	if !ok || len(outward) == 0 {
		return nil, false
	}
	chain := &mixfixChain{
		root:         ident.Name,
		segments:     []string{ident.Name},
		connectors:   []string{""},
		groupSizes:   []int{len(rootArgs)},
		spaced:       []bool{false},
		rootPos:      ident.Position,
		connectorPos: []ast.Position{{}},
	}
	for i := len(outward) - 1; i >= 0; i-- {
		part := outward[i]
		chain.connectors = append(chain.connectors, part.connector)
		chain.segments = append(chain.segments, part.name)
		chain.groupSizes = append(chain.groupSizes, part.size)
		chain.spaced = append(chain.spaced, part.spaced)
		chain.connectorPos = append(chain.connectorPos, part.connectorPos)
	}
	return chain, true
}

// childrenOf 返回 AST 节点的直接子节点（语句与表达式）。
func childrenOf(node ast.Node) []ast.Node {
	var children []ast.Node
	addStmt := func(s ast.Statement) { children = append(children, s) }
	addExpr := func(e ast.Expression) {
		if e != nil {
			children = append(children, e)
		}
	}
	switch n := node.(type) {
	case *ast.Program:
		for _, s := range n.Statements {
			addStmt(s)
		}
	case *ast.BlockStmt:
		for _, s := range n.Statements {
			addStmt(s)
		}
	case *ast.VarDecl:
		addExpr(n.Value)
	case *ast.MultiDecl:
		addExpr(n.Value)
	case *ast.AssignStmt:
		addExpr(n.Value)
	case *ast.CompoundAssignStmt:
		addExpr(n.Target)
		addExpr(n.Value)
	case *ast.VisibilityStmt:
		addStmt(n.Inner)
	case *ast.ExpressionStmt:
		addExpr(n.Expression)
	case *ast.ReturnStmt:
		for _, v := range n.Values {
			addExpr(v)
		}
	case *ast.FunctionLiteral:
		addStmt(n.Body)
		for _, param := range n.Params {
			if param.Default != nil {
				addExpr(param.Default)
			}
		}
	case *ast.CallExpr:
		addExpr(n.Callee)
		for _, a := range n.Args {
			addExpr(a)
		}
	case *ast.MemberExpr:
		addExpr(n.Object)
	case *ast.ImplicitReceiverCallExpr:
		addExpr(n.Receiver)
		for _, a := range n.Args {
			addExpr(a)
		}
	case *ast.MaybeCallExpr:
		addExpr(n.Callee)
	case *ast.BinaryExpr:
		addExpr(n.Left)
		addExpr(n.Right)
	case *ast.TypeTestExpr:
		addExpr(n.Object)
	case *ast.TypeCastExpr:
		addExpr(n.Object)
	case *ast.UnaryExpr:
		addExpr(n.Right)
	case *ast.TernaryExpr:
		addExpr(n.Cond)
		addExpr(n.Then)
		addExpr(n.Else)
	case *ast.IndexExpr:
		addExpr(n.Target)
		addExpr(n.Index)
	case *ast.ArrayLiteral:
		for _, e := range n.Elements {
			addExpr(e)
		}
	case *ast.DictLiteral:
		for _, pair := range n.Pairs {
			addExpr(pair.Key)
			addExpr(pair.Value)
		}
	case *ast.InterpString:
		for _, e := range n.Exprs {
			addExpr(e)
		}
	case *ast.CompositeLiteral:
		// 模型名由 composite-unknown-model 专门诊断，这里只遍历字段值。
		for _, f := range n.Fields {
			addExpr(f.Value)
		}
	case *ast.IfStmt:
		for _, branch := range n.Branches {
			addExpr(branch.Condition)
			addStmt(branch.Body)
		}
		if n.Else != nil {
			addStmt(n.Else)
		}
	case *ast.WhileStmt:
		addExpr(n.Condition)
		addStmt(n.Body)
	case *ast.RepeatStmt:
		addStmt(n.Body)
		addExpr(n.Until)
	case *ast.TimesStmt:
		addExpr(n.Count)
		addStmt(n.Body)
	case *ast.ForEachStmt:
		addExpr(n.Iterable)
		addStmt(n.Body)
	case *ast.SwitchStmt:
		addExpr(n.Subject)
		for _, branch := range n.Branches {
			for _, value := range branch.Values {
				addExpr(value)
			}
			addStmt(branch.Body)
		}
	case *ast.MethodDef:
		addExpr(n.Function)
	case *ast.DeferStmt:
		addExpr(n.Call)
	case *ast.ThrowStmt:
		addExpr(n.Value)
	case *ast.CheckExpr:
		addExpr(n.Call)
	case *ast.ThroughExpr:
		addExpr(n.Expr)
	}
	return children
}

// walk 深度优先遍历 AST。
func walk(node ast.Node, visit func(ast.Node) bool) {
	walkWithParents(node, nil, func(n ast.Node, _ ast.Node) bool {
		return visit(n)
	})
}

// walkWithParents 遍历时向回调提供直接父节点。
func walkWithParents(node ast.Node, parent ast.Node, visit func(ast.Node, ast.Node) bool) {
	if node == nil || !visit(node, parent) {
		return
	}
	for _, child := range childrenOf(node) {
		walkWithParents(child, node, visit)
	}
}

// UTF16Column 把 1-based、以 rune 计的列转换为 LSP 使用的 0-based UTF-16 码元列。
func UTF16Column(line string, runeColumn int) int {
	if runeColumn <= 1 {
		return 0
	}
	runes := []rune(line)
	count := runeColumn - 1
	if count > len(runes) {
		count = len(runes)
	}
	return len(utf16.Encode(runes[:count]))
}
