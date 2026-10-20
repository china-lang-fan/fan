package ast

import (
	"fmt"
	"strings"
)

type ArrayLiteral struct {
	Position Position
	Elements []Expression
}

func (n *ArrayLiteral) Pos() Position   { return n.Position }
func (n *ArrayLiteral) expressionNode() {}
func (n *ArrayLiteral) String() string {
	parts := make([]string, 0, len(n.Elements))
	for _, e := range n.Elements {
		parts = append(parts, e.String())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

type IndexExpr struct {
	Position Position
	Target   Expression
	Index    Expression
}

func (n *IndexExpr) Pos() Position   { return n.Position }
func (n *IndexExpr) expressionNode() {}
func (n *IndexExpr) String() string {
	return fmt.Sprintf("%s[%s]", n.Target.String(), n.Index.String())
}

// SliceExpr 表示切片表达式 目标[低:高]，Low/High 为 nil 表示该侧省略。
type SliceExpr struct {
	Position Position
	Target   Expression
	Low      Expression
	High     Expression
	ColonPos Position
}

func (n *SliceExpr) Pos() Position   { return n.Position }
func (n *SliceExpr) expressionNode() {}
func (n *SliceExpr) String() string {
	low, high := "", ""
	if n.Low != nil {
		low = n.Low.String()
	}
	if n.High != nil {
		high = n.High.String()
	}
	return fmt.Sprintf("%s[%s:%s]", n.Target.String(), low, high)
}

// SpreadExpr 表示展开表达式 ...值，用于数组字面量、调用实参与字典字面量。
type SpreadExpr struct {
	Position Position
	Value    Expression
}

func (n *SpreadExpr) Pos() Position   { return n.Position }
func (n *SpreadExpr) expressionNode() {}
func (n *SpreadExpr) String() string  { return "..." + n.Value.String() }

type CallExpr struct {
	Position Position
	Callee   Expression
	Args     []Expression
}

func (n *CallExpr) Pos() Position   { return n.Position }
func (n *CallExpr) expressionNode() {}
func (n *CallExpr) String() string {
	parts := make([]string, 0, len(n.Args))
	for _, a := range n.Args {
		parts = append(parts, a.String())
	}
	return fmt.Sprintf("%s(%s)", n.Callee.String(), strings.Join(parts, ", "))
}

type IndexAssignStmt struct {
	Position Position
	Target   Expression
	Index    Expression
	Value    Expression
}

func (n *IndexAssignStmt) Pos() Position  { return n.Position }
func (n *IndexAssignStmt) statementNode() {}
func (n *IndexAssignStmt) String() string {
	return fmt.Sprintf("%s[%s] = %s", n.Target.String(), n.Index.String(), n.Value.String())
}

type DictPair struct {
	Key    Expression
	Value  Expression
	Spread bool // {...d} 字典展开时为 true，此时仅 Value 有效
}

type DictLiteral struct {
	Position Position
	Pairs    []DictPair
}

func (n *DictLiteral) Pos() Position   { return n.Position }
func (n *DictLiteral) expressionNode() {}
func (n *DictLiteral) String() string {
	parts := make([]string, 0, len(n.Pairs))
	for _, p := range n.Pairs {
		if p.Spread {
			parts = append(parts, "..."+p.Value.String())
			continue
		}
		parts = append(parts, p.Key.String()+": "+p.Value.String())
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
