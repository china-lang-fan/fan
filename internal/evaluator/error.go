package evaluator

import (
	"fan/internal/ast"
	"fan/internal/object"
)

type checkSignal struct {
	err *object.Error
	pos ast.Position
}

func (c *checkSignal) Error() string { return c.err.Inspect() }

func evalCheckExpr(expr *ast.CheckExpr, env *Environment) (object.Object, error) {
	val, err := Eval(expr.Call, env)
	if err != nil {
		if sig, ok := err.(*checkSignal); ok {
			return nil, sig
		}
		return nil, err
	}
	if e, ok := errorFromValue(val); ok {
		return nil, &checkSignal{err: e, pos: expr.Position}
	}
	return stripErrorTail(val), nil
}

func errorFromValue(val object.Object) (*object.Error, bool) {
	if val == nil {
		return nil, false
	}
	if e, ok := val.(*object.Error); ok {
		return e, true
	}
	if t, ok := val.(*object.Tuple); ok && len(t.Values) > 0 {
		last := t.Values[len(t.Values)-1]
		if e, ok := last.(*object.Error); ok {
			return e, true
		}
	}
	return nil, false
}

func stripErrorTail(val object.Object) object.Object {
	if t, ok := val.(*object.Tuple); ok && len(t.Values) > 1 {
		last := t.Values[len(t.Values)-1]
		if last.Kind() == object.KindError || last.Kind() == object.KindNil {
			values := t.Values[:len(t.Values)-1]
			if len(values) == 1 {
				return values[0]
			}
			return &object.Tuple{Values: values}
		}
	}
	return val
}

func unwrapCheckSignal(err error) (*object.Error, bool) {
	if sig, ok := err.(*checkSignal); ok {
		return sig.err, true
	}
	return nil, false
}

func zeroValueForType(dt ast.DeclType) object.Object {
	switch dt {
	case ast.TypeInt:
		return &object.Integer{}
	case ast.TypeFloat:
		return &object.Float{}
	case ast.TypeString:
		return &object.String{}
	case ast.TypeBool:
		return object.False
	case ast.TypeArray:
		return &object.Array{}
	case ast.TypeDict:
		return &object.Dict{}
	case ast.TypeError:
		return object.Null
	}
	return object.Null
}
