package evaluator

import (
	"fmt"

	"fan/internal/ast"
	"fan/internal/object"
)

func evalArrayLiteral(node *ast.ArrayLiteral, env *Environment) (object.Object, error) {
	elems := make([]object.Object, 0, len(node.Elements))
	for _, el := range node.Elements {
		if spread, ok := el.(*ast.SpreadExpr); ok {
			val, err := evalSpread(spread, env, "数组字面量")
			if err != nil {
				return nil, err
			}
			elems = append(elems, val.(*spreadMarker).values...)
			continue
		}
		val, err := Eval(el, env)
		if err != nil {
			return nil, err
		}
		elems = append(elems, val)
	}
	return &object.Array{Elements: elems}, nil
}

func evalDictLiteral(node *ast.DictLiteral, env *Environment) (object.Object, error) {
	dict := object.NewDict()
	for _, pair := range node.Pairs {
		if pair.Spread {
			val, err := Eval(pair.Value, env)
			if err != nil {
				return nil, err
			}
			src, ok := val.(*object.Dict)
			if !ok {
				return nil, &EvalError{Pos: pair.Value.Pos(), Reason: fmt.Sprintf("字典展开 ... 只能展开字典，实际是 %s", val.Kind())}
			}
			for _, k := range src.Keys {
				v, _ := src.Get(k)
				dict.Set(k, v)
			}
			continue
		}
		key, err := Eval(pair.Key, env)
		if err != nil {
			return nil, err
		}
		if !object.ValidDictKey(key) {
			return nil, &EvalError{Pos: pair.Key.Pos(), Reason: fmt.Sprintf("字典键必须是整数/小数/字符串/布尔，实际是 %s", key.Kind())}
		}
		val, err := Eval(pair.Value, env)
		if err != nil {
			return nil, err
		}
		dict.Set(key, val)
	}
	return dict, nil
}

func evalIndexExpr(node *ast.IndexExpr, env *Environment) (object.Object, error) {
	target, err := Eval(node.Target, env)
	if err != nil {
		return nil, err
	}
	idx, err := Eval(node.Index, env)
	if err != nil {
		return nil, err
	}
	switch t := target.(type) {
	case *object.Array:
		i, err := indexValue(node.Position, idx)
		if err != nil {
			return nil, err
		}
		if i < 0 || i >= int64(len(t.Elements)) {
			return nil, &EvalError{
				Pos:    node.Position,
				Reason: fmt.Sprintf("数组下标越界：长度 %d，下标 %d", len(t.Elements), i),
			}
		}
		return t.Elements[i], nil
	case *object.Dict:
		if !object.ValidDictKey(idx) {
			return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("字典键必须是整数/小数/字符串/布尔，实际是 %s", idx.Kind())}
		}
		if v, ok := t.Get(idx); ok {
			return v, nil
		}
		return object.Null, nil
	}
	return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 不支持下标访问", target.Kind())}
}

func evalIndexAssign(node *ast.IndexAssignStmt, env *Environment) (object.Object, error) {
	target, err := Eval(node.Target, env)
	if err != nil {
		return nil, err
	}
	idx, err := Eval(node.Index, env)
	if err != nil {
		return nil, err
	}
	val, err := Eval(node.Value, env)
	if err != nil {
		return nil, err
	}
	switch t := target.(type) {
	case *object.Array:
		i, err := indexValue(node.Position, idx)
		if err != nil {
			return nil, err
		}
		if i < 0 || i >= int64(len(t.Elements)) {
			return nil, &EvalError{
				Pos:    node.Position,
				Reason: fmt.Sprintf("数组下标越界：长度 %d，下标 %d", len(t.Elements), i),
			}
		}
		t.Elements[i] = val
		return val, nil
	case *object.Dict:
		if !object.ValidDictKey(idx) {
			return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("字典键必须是整数/小数/字符串/布尔，实际是 %s", idx.Kind())}
		}
		t.Set(idx, val)
		return val, nil
	}
	return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 不支持下标赋值", target.Kind())}
}

func indexValue(pos ast.Position, idx object.Object) (int64, error) {
	n, ok := idx.(*object.Integer)
	if !ok {
		return 0, &EvalError{Pos: pos, Reason: fmt.Sprintf("下标必须是整数，实际是 %s", idx.Kind())}
	}
	return n.Value, nil
}

func evalArrayConcat(pos ast.Position, left, right *object.Array) object.Object {
	elems := make([]object.Object, 0, len(left.Elements)+len(right.Elements))
	elems = append(elems, left.Elements...)
	elems = append(elems, right.Elements...)
	return &object.Array{Elements: elems}
}

func dictEqual(a, b *object.Dict) bool {
	if a.Len() != b.Len() {
		return false
	}
	for _, k := range a.Keys {
		av, aok := a.Get(k)
		bv, bok := b.Get(k)
		if !aok || !bok {
			return false
		}
		if !object.DeepEqual(av, bv) {
			return false
		}
	}
	return true
}

func evalDictMember(node *ast.MemberExpr, dict *object.Dict) (object.Object, error) {
	switch node.Name {
	case "长度":
		return &object.Integer{Value: int64(dict.Len())}, nil
	case "键":
		return &object.Array{Elements: append([]object.Object(nil), dict.Keys...)}, nil
	case "值":
		vals := make([]object.Object, 0, dict.Len())
		for _, k := range dict.Keys {
			v, _ := dict.Get(k)
			vals = append(vals, v)
		}
		return &object.Array{Elements: vals}, nil
	}
	return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("字典没有成员 %s", node.Name)}
}
