package ast

import (
	"fmt"
	"strings"
)

type DeclType string

const (
	TypeAny    DeclType = ""
	TypeInt    DeclType = "整数"
	TypeFloat  DeclType = "小数"
	TypeString DeclType = "字符串"
	TypeBool   DeclType = "布尔"
	TypeArray  DeclType = "数组"
	TypeDict   DeclType = "字典"
	TypeError  DeclType = "错误"
	TypeByte   DeclType = "字节"
	TypeChar   DeclType = "字符"
)

type Position struct {
	Line   int
	Column int
}

type Node interface {
	Pos() Position
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) Pos() Position { return Position{Line: 1, Column: 1} }
func (p *Program) String() string {
	var b strings.Builder
	for _, s := range p.Statements {
		b.WriteString(s.String())
		b.WriteString("\n")
	}
	return b.String()
}

type VarDecl struct {
	Position   Position
	IsConst    bool
	IsExplicit bool
	DeclType   DeclType
	Name       string
	Value      Expression
}

func (v *VarDecl) Pos() Position  { return v.Position }
func (v *VarDecl) statementNode() {}
func (v *VarDecl) String() string {
	kind := "变量"
	if v.IsConst {
		kind = "常量"
	}
	typ := ""
	if v.DeclType != TypeAny {
		typ = " " + string(v.DeclType)
	}
	return fmt.Sprintf("%s%s %s = %s", kind, typ, v.Name, v.Value.String())
}

type AssignStmt struct {
	Position Position
	Name     string
	Value    Expression
}

func (a *AssignStmt) Pos() Position  { return a.Position }
func (a *AssignStmt) statementNode() {}
func (a *AssignStmt) String() string {
	return fmt.Sprintf("%s = %s", a.Name, a.Value.String())
}

type CompoundAssignStmt struct {
	Position Position
	Target   Expression
	Op       string
	Value    Expression
}

func (a *CompoundAssignStmt) Pos() Position  { return a.Position }
func (a *CompoundAssignStmt) statementNode() {}
func (a *CompoundAssignStmt) String() string {
	return fmt.Sprintf("(%s %s %s)", a.Target.String(), a.Op, a.Value.String())
}

type UpdateExpr struct {
	Position Position
	Target   Expression
	Op       string
	Prefix   bool
}

func (a *UpdateExpr) Pos() Position   { return a.Position }
func (a *UpdateExpr) expressionNode() {}
func (a *UpdateExpr) String() string {
	if a.Prefix {
		return fmt.Sprintf("(%s%s)", a.Op, a.Target.String())
	}
	return fmt.Sprintf("(%s%s)", a.Target.String(), a.Op)
}

type MultiDecl struct {
	Position   Position
	IsConst    bool
	IsExplicit bool
	Names      []string
	Value      Expression
}

func (m *MultiDecl) Pos() Position  { return m.Position }
func (m *MultiDecl) statementNode() {}
func (m *MultiDecl) String() string {
	return fmt.Sprintf("变量 %s = %s", strings.Join(m.Names, "、"), m.Value.String())
}

type MultiAssign struct {
	Position Position
	Names    []string
	Value    Expression
}

func (m *MultiAssign) Pos() Position  { return m.Position }
func (m *MultiAssign) statementNode() {}
func (m *MultiAssign) String() string {
	return fmt.Sprintf("%s = %s", strings.Join(m.Names, "、"), m.Value.String())
}

type ExpressionStmt struct {
	Position   Position
	Expression Expression
}

func (e *ExpressionStmt) Pos() Position  { return e.Position }
func (e *ExpressionStmt) statementNode() {}
func (e *ExpressionStmt) String() string {
	return e.Expression.String()
}

type IntegerLiteral struct {
	Position Position
	Value    int64
}

func (n *IntegerLiteral) Pos() Position   { return n.Position }
func (n *IntegerLiteral) expressionNode() {}
func (n *IntegerLiteral) String() string  { return fmt.Sprintf("%d", n.Value) }

type FloatLiteral struct {
	Position Position
	Value    float64
}

func (n *FloatLiteral) Pos() Position   { return n.Position }
func (n *FloatLiteral) expressionNode() {}
func (n *FloatLiteral) String() string  { return fmt.Sprintf("%g", n.Value) }

type StringLiteral struct {
	Position Position
	Value    string
}

func (n *StringLiteral) Pos() Position   { return n.Position }
func (n *StringLiteral) expressionNode() {}
func (n *StringLiteral) String() string  { return fmt.Sprintf("%q", n.Value) }

type CharLiteral struct {
	Position Position
	Value    rune
}

func (n *CharLiteral) Pos() Position   { return n.Position }
func (n *CharLiteral) expressionNode() {}
func (n *CharLiteral) String() string  { return fmt.Sprintf("%q", string(n.Value)) }

type BoolLiteral struct {
	Position Position
	Value    bool
}

func (n *BoolLiteral) Pos() Position   { return n.Position }
func (n *BoolLiteral) expressionNode() {}
func (n *BoolLiteral) String() string {
	if n.Value {
		return "真"
	}
	return "假"
}

type NilLiteral struct {
	Position Position
}

func (n *NilLiteral) Pos() Position   { return n.Position }
func (n *NilLiteral) expressionNode() {}
func (n *NilLiteral) String() string  { return "空" }

type Identifier struct {
	Position Position
	Name     string
}

func (n *Identifier) Pos() Position   { return n.Position }
func (n *Identifier) expressionNode() {}
func (n *Identifier) String() string  { return n.Name }

type BlockStmt struct {
	Position   Position
	Statements []Statement
}

func (b *BlockStmt) Pos() Position  { return b.Position }
func (b *BlockStmt) statementNode() {}
func (b *BlockStmt) String() string {
	var sb strings.Builder
	for _, s := range b.Statements {
		sb.WriteString(s.String())
		sb.WriteString("; ")
	}
	return strings.TrimSuffix(sb.String(), " ")
}

type IfBranch struct {
	Condition Expression
	Body      *BlockStmt
}

type IfStmt struct {
	Position Position
	Branches []IfBranch
	Else     *BlockStmt
}

func (i *IfStmt) Pos() Position  { return i.Position }
func (i *IfStmt) statementNode() {}
func (i *IfStmt) String() string {
	var sb strings.Builder
	for idx, br := range i.Branches {
		if idx == 0 {
			sb.WriteString("如果 ")
		} else {
			sb.WriteString("否则如果 ")
		}
		sb.WriteString(br.Condition.String())
		sb.WriteString(" 那么 { ")
		sb.WriteString(br.Body.String())
		sb.WriteString(" } ")
	}
	if i.Else != nil {
		sb.WriteString("否则 { ")
		sb.WriteString(i.Else.String())
		sb.WriteString(" } ")
	}
	sb.WriteString("结束")
	return sb.String()
}

type UnaryExpr struct {
	Position Position
	Op       string
	Right    Expression
}

func (n *UnaryExpr) Pos() Position   { return n.Position }
func (n *UnaryExpr) expressionNode() {}
func (n *UnaryExpr) String() string {
	sep := ""
	if n.Op == "-" || n.Op == "!" {
		sep = ""
	} else {
		sep = " "
	}
	return fmt.Sprintf("(%s%s%s)", n.Op, sep, n.Right.String())
}

type MaybeCallExpr struct {
	Position Position
	Callee   Expression
	AutoCall bool
}

func (n *MaybeCallExpr) Pos() Position   { return n.Position }
func (n *MaybeCallExpr) expressionNode() {}
func (n *MaybeCallExpr) String() string {
	return n.Callee.String()
}

type TernaryExpr struct {
	Position Position
	Cond     Expression
	Then     Expression
	Else     Expression
}

func (n *TernaryExpr) Pos() Position   { return n.Position }
func (n *TernaryExpr) expressionNode() {}
func (n *TernaryExpr) String() string {
	return fmt.Sprintf("(%s ? %s : %s)", n.Cond.String(), n.Then.String(), n.Else.String())
}

type BinaryExpr struct {
	Position Position
	Op       string
	Left     Expression
	Right    Expression
}

func (n *BinaryExpr) Pos() Position   { return n.Position }
func (n *BinaryExpr) expressionNode() {}
func (n *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left.String(), n.Op, n.Right.String())
}
