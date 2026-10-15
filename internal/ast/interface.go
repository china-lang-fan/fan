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
