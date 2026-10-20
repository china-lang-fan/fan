package evaluator

import (
	"fmt"
	"strings"

	"fan/internal/ast"
	"fan/internal/object"
)

const KindNamedType object.Kind = "自定义类型"

// NamedType 是“类型 名 底层类型”声明在运行时的类型对象。
// 作为值调用时执行显式转换：状态(1) -> *object.Named{TypeName:"状态",...}
type NamedType struct {
	Name     string
	BaseType ast.DeclType
	IsUnion  bool
	Methods  map[string]*Function
}

func (t *NamedType) Kind() object.Kind { return KindNamedType }
func (t *NamedType) Inspect() string   { return fmt.Sprintf("<类型 %s = %s>", t.Name, t.BaseType) }

func (t *NamedType) lookupMethod(name string) *Function {
	return t.Methods[name]
}

func evalTypeDefStmt(stmt *ast.TypeDefStmt, env *Environment) (object.Object, error) {
	rt := env.Runtime()
	isUnion := strings.Contains(string(stmt.BaseType), "|")
	named := &NamedType{Name: stmt.Name, BaseType: stmt.BaseType, IsUnion: isUnion, Methods: map[string]*Function{}}
	rt.namedTypes[stmt.Name] = named
	if existing, ok := env.get(stmt.Name); ok {
		if old, ok := existing.(*NamedType); ok {
			old.BaseType = stmt.BaseType
			old.IsUnion = isUnion
			if old.Methods == nil {
				old.Methods = map[string]*Function{}
			}
			return old, nil
		}
	}
	if err := env.declare(stmt.Name, named, true, ast.TypeAny); err != nil {
		return nil, &EvalError{Pos: stmt.Position, Reason: err.Error()}
	}
	return named, nil
}

// convertToNamed 把值收窄并包装为自定义类型值。
// 联合命名类型：值满足任一成员即通过；字节成员按 0–255 收窄。
func convertToNamed(t *NamedType, value object.Object, pos ast.Position) (object.Object, error) {
	if t.IsUnion {
		return convertToUnionNamed(t, value, pos)
	}
	if named, ok := value.(*object.Named); ok {
		value = named.Value
	}
	converted, err := coerceToPrimitiveKind(value, t.BaseType)
	if err != nil {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("无法转换为 %s：%s", t.Name, err.Error())}
	}
	return &object.Named{TypeName: t.Name, Value: converted}, nil
}

// convertToUnionNamed 校验值是否满足联合命名类型，并包装为该命名类型值。
func convertToUnionNamed(t *NamedType, value object.Object, pos ast.Position) (object.Object, error) {
	inner := value
	if named, ok := value.(*object.Named); ok {
		inner = named.Value
	}
	if member, ok := unionMatchMember(t.BaseType, inner, nil); ok {
		var converted object.Object = inner
		if member == ast.TypeByte {
			if b, ok := inner.(*object.Byte); ok {
				converted = b
			} else if n, ok := inner.(*object.Integer); ok && n.Value >= 0 && n.Value <= 255 {
				converted = &object.Byte{Value: byte(n.Value)}
			}
		}
		return &object.Named{TypeName: t.Name, Value: converted}, nil
	}
	return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("无法转换为 %s：类型不匹配（%s），实际为 %s", t.Name, formatDeclType(t.BaseType), value.Kind())}
}

// unionMatchMember 返回值满足的第一个联合成员名；env 非空时支持接口成员。
func unionMatchMember(dt ast.DeclType, value object.Object, env *Environment) (ast.DeclType, bool) {
	if !strings.Contains(string(dt), "|") {
		if checkSingleDeclType(dt, value, env) == nil {
			return dt, true
		}
		return "", false
	}
	for _, member := range strings.Split(string(dt), "|") {
		if member == "空" {
			if value.Kind() == object.KindNil {
				return ast.DeclType(member), true
			}
			continue
		}
		if checkSingleDeclType(ast.DeclType(member), value, env) == nil {
			return ast.DeclType(member), true
		}
	}
	return "", false
}

// coerceToPrimitiveKind 按声明的基本类型收窄值；字节允许 0–255 整数自动转换。
func coerceToPrimitiveKind(value object.Object, dt ast.DeclType) (object.Object, error) {
	switch dt {
	case ast.TypeByte:
		if b, ok := value.(*object.Byte); ok {
			return b, nil
		}
		if n, ok := value.(*object.Integer); ok {
			if n.Value < 0 || n.Value > 255 {
				return nil, fmt.Errorf("字节超出范围：%d（允许 0–255）", n.Value)
			}
			return &object.Byte{Value: byte(n.Value)}, nil
		}
	default:
		if err := checkDeclType(dt, value, nil); err == nil {
			return value, nil
		}
	}
	// 宽松值转换：字符串解析、数值互转等。
	if converted, ok := convertToPrimitiveKind(value, declTypeToRuntimeKind(dt)); ok {
		return converted, nil
	}
	return nil, fmt.Errorf("类型不匹配：需要 %s，实际为 %s", dt, value.Kind())
}

func declTypeToRuntimeKind(dt ast.DeclType) string {
	if kind, ok := primitiveKindFromName(string(dt)); ok {
		return kind
	}
	return ""
}

// namedTypeName 返回自定义类型值的类型名，非自定义值返回空串。
func namedTypeName(value object.Object) string {
	if named, ok := value.(*object.Named); ok {
		return named.TypeName
	}
	return ""
}

// unwrapNamed 解包自定义类型，返回底层值。
func unwrapNamed(value object.Object) object.Object {
	if named, ok := value.(*object.Named); ok {
		return named.Value
	}
	return value
}
