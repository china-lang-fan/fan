package object

import (
	"fmt"
	"strconv"
)

type Kind string

const (
	KindInt    Kind = "整数"
	KindFloat  Kind = "小数"
	KindString Kind = "字符串"
	KindBool   Kind = "布尔"
	KindByte   Kind = "字节"
	KindChar   Kind = "字符"
	KindNil    Kind = "空"
)

type Object interface {
	Kind() Kind
	Inspect() string
}

type Integer struct{ Value int64 }

func (i *Integer) Kind() Kind      { return KindInt }
func (i *Integer) Inspect() string { return strconv.FormatInt(i.Value, 10) }

type Float struct{ Value float64 }

func (f *Float) Kind() Kind { return KindFloat }
func (f *Float) Inspect() string {
	return strconv.FormatFloat(f.Value, 'g', -1, 64)
}

type String struct{ Value string }

func (s *String) Kind() Kind      { return KindString }
func (s *String) Inspect() string { return s.Value }

// Byte 是 0–255 的 8 位无符号整数。
type Byte struct{ Value byte }

func (b *Byte) Kind() Kind      { return KindByte }
func (b *Byte) Inspect() string { return strconv.Itoa(int(b.Value)) }

// Char 是单个 Unicode 字符。
type Char struct{ Value rune }

func (c *Char) Kind() Kind      { return KindChar }
func (c *Char) Inspect() string { return string(c.Value) }

type Bool struct{ Value bool }

func (b *Bool) Kind() Kind { return KindBool }
func (b *Bool) Inspect() string {
	if b.Value {
		return "真"
	}
	return "假"
}

// Named 包装一个基本类型值，携带用户用“类型 名 底层类型”定义的类型名，
// 参考 Go 中 type Status int 的身份语义：运算时按底层类型参与，
// 类型判断/赋值检查时保留独立类型身份。
type Named struct {
	TypeName string
	Value    Object
}

func (n *Named) Kind() Kind      { return n.Value.Kind() }
func (n *Named) Inspect() string { return n.Value.Inspect() }

type Nil struct{}

func (n *Nil) Kind() Kind      { return KindNil }
func (n *Nil) Inspect() string { return "空" }

var (
	True  = &Bool{Value: true}
	False = &Bool{Value: false}
	Null  = &Nil{}
)

func BoolOf(b bool) *Bool {
	if b {
		return True
	}
	return False
}

func Format(o Object) string {
	if o == nil {
		return "空"
	}
	return o.Inspect()
}

func Debug(o Object) string {
	return fmt.Sprintf("%s(%s)", o.Kind(), o.Inspect())
}
