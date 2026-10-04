package ast

import (
	"fmt"
	"strings"
)

type TagExpr struct {
	Position   Position
	TypeName   string
	Positional []Expression
	Named      []NamedTagArg
}

type NamedTagArg struct {
	Name  string
	Value Expression
}

func (n *TagExpr) Pos() Position   { return n.Position }
func (n *TagExpr) expressionNode() {}

func (n *TagExpr) String() string {
	var sb strings.Builder
	sb.WriteString("@" + n.TypeName)
	if len(n.Positional) == 0 && len(n.Named) == 0 {
		return sb.String()
	}
	sb.WriteString("(")
	for i, expr := range n.Positional {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(expr.String())
	}
	for i, arg := range n.Named {
		if i > 0 || len(n.Positional) > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(arg.Name + " = " + arg.Value.String())
	}
	sb.WriteString(")")
	return sb.String()
}

var _ Node = (*TagExpr)(nil)

func TagNames(tags []*TagExpr) []string {
	names := make([]string, len(tags))
	for i, tag := range tags {
		names[i] = fmt.Sprintf("@%s", tag.TypeName)
	}
	return names
}
