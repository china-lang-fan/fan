package ast

import "fmt"

type ImportStmt struct {
	Position Position
	Path     string
	Name     string
}

func (n *ImportStmt) Pos() Position  { return n.Position }
func (n *ImportStmt) statementNode() {}
func (n *ImportStmt) String() string {
	return fmt.Sprintf("导入 %q 作为 %s", n.Path, n.Name)
}

// ThroughExpr 是“通过 模块名 表达式”：在指定模块作用域内求值表达式，
// 只能访问该模块的公开成员。
type ThroughExpr struct {
	Position   Position
	ModuleName string
	Expr       Expression
}

func (n *ThroughExpr) Pos() Position   { return n.Position }
func (n *ThroughExpr) expressionNode() {}
func (n *ThroughExpr) String() string {
	return fmt.Sprintf("通过 %s %s", n.ModuleName, n.Expr.String())
}

// Visibility 表示声明的可见性。
type Visibility int

const (
	// VisFile 是默认可见性：仅当前文件（模块）内可访问。
	VisFile Visibility = iota
	// VisPublic 表示公开：可跨模块访问（公开 关键字）。
	VisPublic
	// VisPrivate 表示私有：模块级等同文件私有；模型字段/方法表示仅本模型方法可访问。
	VisPrivate
)

// VisibilityStmt 是 公开/私有 修饰的声明：
//
//	公开 函数 读取(...) ...
type VisibilityStmt struct {
	Position   Position
	Name       string
	Visibility Visibility
	Inner      Statement
}

func (n *VisibilityStmt) Pos() Position  { return n.Position }
func (n *VisibilityStmt) statementNode() {}
func (n *VisibilityStmt) String() string {
	kw := "私有"
	if n.Visibility == VisPublic {
		kw = "公开"
	}
	return fmt.Sprintf("%s %s", kw, n.Inner.String())
}
