package ast

import (
	"fmt"
	"strings"
)

type FieldDecl struct {
	Name string
	Type DeclType
}

type ClassStmt struct {
	Position Position
	Name     string
	Fields   []FieldDecl
	Embeds   []string
	Tags     []*TagExpr
}

type MethodDef struct {
	Position   Position
	ClassName  string
	MethodName string
	Function   *FunctionLiteral
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
