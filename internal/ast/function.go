package ast

import (
	"fmt"
	"strings"
)

type Parameter struct {
	Name     string
	Type     DeclType
	Variadic bool
	Default  Expression
}

type FunctionLiteral struct {
	Position    Position
	Params      []Parameter
	Body        *BlockStmt
	Name        string
	ReturnTypes []DeclType
	Tags        []*TagExpr
}

func (n *FunctionLiteral) Pos() Position   { return n.Position }
func (n *FunctionLiteral) expressionNode() {}
func (n *FunctionLiteral) String() string {
	var sb strings.Builder
	if n.Name != "" {
		sb.WriteString("函数 " + n.Name)
	} else {
		sb.WriteString("函数")
	}
	sb.WriteString("(")
	for i, p := range n.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		if p.Variadic {
			if p.Type != TypeAny {
				sb.WriteString(string(p.Type) + " ")
			}
			sb.WriteString("..." + p.Name)
		} else {
			sb.WriteString(p.Name)
			if p.Type != TypeAny {
				sb.WriteString(" " + string(p.Type))
			}
			if p.Default != nil {
				sb.WriteString(" = " + p.Default.String())
			}
		}
	}
	sb.WriteString(") ")
	if len(n.ReturnTypes) > 0 {
		sb.WriteString("返回 ")
		for i, dt := range n.ReturnTypes {
			if i > 0 {
				sb.WriteString("、")
			}
			if dt == TypeAny {
				sb.WriteString("任意")
			} else {
				sb.WriteString(string(dt))
			}
		}
		sb.WriteString(" ")
	}
	sb.WriteString(n.Body.String())
	sb.WriteString(" 结束")
	return sb.String()
}

type ReturnStmt struct {
	Position Position
	Values   []Expression
}

func (n *ReturnStmt) Pos() Position  { return n.Position }
func (n *ReturnStmt) statementNode() {}
func (n *ReturnStmt) String() string {
	if len(n.Values) == 0 {
		return "返回"
	}
	parts := make([]string, len(n.Values))
	for i, v := range n.Values {
		parts[i] = v.String()
	}
	return fmt.Sprintf("返回 %s", strings.Join(parts, "、"))
}

type MemberExpr struct {
	Position Position
	Object   Expression
	Name     string
}

func (n *MemberExpr) Pos() Position   { return n.Position }
func (n *MemberExpr) expressionNode() {}
func (n *MemberExpr) String() string {
	return fmt.Sprintf("%s 的 %s", n.Object.String(), n.Name)
}
