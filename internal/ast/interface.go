package ast

// InterfaceMethodSig 是接口中的方法签名声明（无函数体）。
type InterfaceMethodSig struct {
	Name        string
	Params      []Parameter
	ReturnTypes []DeclType
	Position    Position
}

// InterfaceStmt 声明一个接口：
//
//	接口 说话者
//	    方法 说话() 字符串
//	结束
type InterfaceStmt struct {
	Position Position
	Name     string
	Embeds   []string
	Methods  []*InterfaceMethodSig
}

func (s *InterfaceStmt) Pos() Position  { return s.Position }
func (s *InterfaceStmt) statementNode() {}
func (s *InterfaceStmt) String() string {
	return "接口 " + s.Name
}

// TypeDefStmt 声明一个自定义类型（参考 Go 的 type 定义）：
//
//	类型 状态 整数
type TypeDefStmt struct {
	Position Position
	Name     string
	BaseType DeclType
}

func (s *TypeDefStmt) Pos() Position  { return s.Position }
func (s *TypeDefStmt) statementNode() {}
func (s *TypeDefStmt) String() string {
	return "类型 " + s.Name
}

// TypeTestExpr 是 “值 是 类型” 表达式，等价于 Java 的 instanceof。
type TypeTestExpr struct {
	Position Position
	Object   Expression
	TypeName string
}

func (e *TypeTestExpr) Pos() Position   { return e.Position }
func (e *TypeTestExpr) expressionNode() {}
func (e *TypeTestExpr) String() string  { return e.TypeName }

// TypeCastExpr 是 “值 转为 类型” 表达式，求值结果为 (值, 是否成功)。
type TypeCastExpr struct {
	Position Position
	Object   Expression
	TypeName string
}

func (e *TypeCastExpr) Pos() Position   { return e.Position }
func (e *TypeCastExpr) expressionNode() {}
func (e *TypeCastExpr) String() string  { return e.TypeName }
