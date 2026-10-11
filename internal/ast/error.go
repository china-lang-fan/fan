package ast

import "fmt"

type CheckExpr struct {
	Position Position
	Call     Expression
}

func (n *CheckExpr) Pos() Position   { return n.Position }
func (n *CheckExpr) expressionNode() {}
func (n *CheckExpr) String() string {
	return fmt.Sprintf("检查 %s", n.Call.String())
}

type DeferStmt struct {
	Position Position
	Call     Expression
}

func (n *DeferStmt) Pos() Position  { return n.Position }
func (n *DeferStmt) statementNode() {}
func (n *DeferStmt) String() string {
	return fmt.Sprintf("稍后 %s", n.Call.String())
}

type ThrowStmt struct {
	Position Position
	Value    Expression
}

func (n *ThrowStmt) Pos() Position  { return n.Position }
func (n *ThrowStmt) statementNode() {}
func (n *ThrowStmt) String() string {
	return fmt.Sprintf("抛出 %s", n.Value.String())
}
