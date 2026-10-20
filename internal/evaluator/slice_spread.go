package evaluator

import (
	"fmt"

	"fan/internal/ast"
	"fan/internal/object"
)

// spreadMarker 标记调用实参中的 ...数组，求值时摊平为位置参数。
type spreadMarker struct {
	values []object.Object
}

func (s *spreadMarker) Kind() object.Kind { return object.KindArray }
func (s *spreadMarker) Inspect() string   { return "展开(...)" }

// evalSlice 求值切片：数组按元素、字符串按 rune。
func evalSlice(node *ast.SliceExpr, env *Environment) (object.Object, error) {
	target, err := Eval(node.Target, env)
	if err != nil {
		return nil, err
	}
	switch t := target.(type) {
	case *object.Array:
		length := int64(len(t.Elements))
		low, high, err := resolveSliceBounds(node, env, length)
		if err != nil {
			return nil, err
		}
		elems := append([]object.Object(nil), t.Elements[low:high]...)
		return &object.Array{Elements: elems}, nil
	case *object.String:
		runes := []rune(t.Value)
		length := int64(len(runes))
		low, high, err := resolveSliceBounds(node, env, length)
		if err != nil {
			return nil, err
		}
		return &object.String{Value: string(runes[low:high])}, nil
	}
	return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 不支持切片", target.Kind())}
}

func resolveSliceBounds(node *ast.SliceExpr, env *Environment, length int64) (int64, int64, error) {
	low := int64(0)
	high := length
	var err error
	if node.Low != nil {
		low, err = evalBound(node.Low, env)
		if err != nil {
			return 0, 0, err
		}
	}
	if node.High != nil {
		high, err = evalBound(node.High, env)
		if err != nil {
			return 0, 0, err
		}
	}
	if low < 0 || high < 0 || low > high || high > length {
		return 0, 0, &EvalError{
			Pos:    node.ColonPos,
			Reason: fmt.Sprintf("切片越界：长度 %d，切片 [%d:%d]", length, low, high),
		}
	}
	return low, high, nil
}

func evalBound(expr ast.Expression, env *Environment) (int64, error) {
	v, err := Eval(expr, env)
	if err != nil {
		return 0, err
	}
	n, ok := v.(*object.Integer)
	if !ok {
		return 0, &EvalError{Pos: expr.Pos(), Reason: fmt.Sprintf("切片边界必须是整数，实际是 %s", v.Kind())}
	}
	return n.Value, nil
}

// evalSpread 展开表达式：数组字面量上下文摊平元素；
// 调用实参上下文返回 spreadMarker；字典展开在字典求值中单独处理。
func evalSpread(node *ast.SpreadExpr, env *Environment, context string) (object.Object, error) {
	val, err := Eval(node.Value, env)
	if err != nil {
		return nil, err
	}
	arr, ok := val.(*object.Array)
	if !ok {
		return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("... 只能展开数组（%s），实际是 %s", context, val.Kind())}
	}
	return &spreadMarker{values: arr.Elements}, nil
}

// flattenSpreadMarkers 把调用实参中的展开标记摊平为位置参数。
func flattenSpreadMarkers(args []object.Object) []object.Object {
	out := make([]object.Object, 0, len(args))
	for _, a := range args {
		if marker, ok := a.(*spreadMarker); ok {
			out = append(out, marker.values...)
			continue
		}
		out = append(out, a)
	}
	return out
}
