package ast

import (
	"fmt"
	"strings"
)

type FieldDecl struct {
	Position   Position
	Name       string
	Type       DeclType
	Visibility Visibility
}

type ClassStmt struct {
	Position Position
	Name     string
	Fields   []FieldDecl
	Embeds   []string
	Tags     []*TagExpr
}

type MethodDef struct {
	Position     Position
	ClassName    string
	MethodName   string
	Function     *FunctionLiteral
	Primitive    bool
	ReceiverName string
	Visibility   Visibility
}

func (n *ClassStmt) Pos() Position  { return n.Position }
func (n *ClassStmt) statementNode() {}
func (n *ClassStmt) String() string {
	var sb strings.Builder
	sb.WriteString("类 " + n.Name + " ")
	for _, f := range n.Fields {
		if f.Type != TypeAny {
			sb.WriteString("变量 " + string(f.Type) + " " + f.Name + " ")
		} else {
			sb.WriteString("变量 " + f.Name + " ")
		}
	}
	for _, e := range n.Embeds {
		sb.WriteString("嵌入 " + e + " ")
	}
	sb.WriteString("结束")
	return sb.String()
}

func (n *MethodDef) Pos() Position  { return n.Position }
func (n *MethodDef) statementNode() {}
func (n *MethodDef) String() string {
	return fmt.Sprintf("定义 %s 的 方法 %s %s", n.ClassName, n.MethodName, n.Function.String())
}

type FieldAssignExpr struct {
	Position Position
	Object   Expression
	Name     string
	Value    Expression
}

func (n *FieldAssignExpr) Pos() Position  { return n.Position }
func (n *FieldAssignExpr) statementNode() {}
func (n *FieldAssignExpr) String() string {
	return fmt.Sprintf("%s.%s = %s", n.Object.String(), n.Name, n.Value.String())
}

// CompositeField 是复合字面量中的一个具名字段初始化项。
type CompositeField struct {
	Name    string
	NamePos Position
	Value   Expression
}

// CompositeLiteral 表示模型复合字面量，如 矩形{宽: 3, 高: 4}。
// Model 通常是 *Identifier，跨模块时为 *MemberExpr（工具.矩形）。
type CompositeLiteral struct {
	Position Position
	Model    Expression
	Fields   []CompositeField
}

func (n *CompositeLiteral) Pos() Position   { return n.Position }
func (n *CompositeLiteral) expressionNode() {}
func (n *CompositeLiteral) String() string {
	var sb strings.Builder
	sb.WriteString(n.Model.String())
	sb.WriteString("{")
	for i, f := range n.Fields {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(f.Name)
		sb.WriteString(": ")
		if f.Value != nil {
			sb.WriteString(f.Value.String())
		}
	}
	sb.WriteString("}")
	return sb.String()
}
