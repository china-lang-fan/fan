package evaluator

import (
	"fmt"

	"fan/internal/ast"
	"fan/internal/object"
)

type assignTarget interface {
	get() (object.Object, error)
	set(object.Object) error
}

type identTarget struct {
	name string
	env  *Environment
	pos  ast.Position
}

func (t *identTarget) get() (object.Object, error) {
	if val, ok := t.env.get(t.name); ok {
		return val, nil
	}
	return nil, &EvalError{Pos: t.pos, Reason: fmt.Sprintf("变量 %s 未声明", t.name)}
}

func (t *identTarget) set(val object.Object) error {
	if err := t.env.assign(t.name, val); err != nil {
		return &EvalError{Pos: t.pos, Reason: err.Error()}
	}
	return nil
}

type indexTarget struct {
	target object.Object
	index  object.Object
	pos    ast.Position
}

func (t *indexTarget) get() (object.Object, error) {
	switch target := t.target.(type) {
	case *object.Array:
		i, ok := t.index.(*object.Integer)
		if !ok {
			return nil, &EvalError{Pos: t.pos, Reason: fmt.Sprintf("下标必须是整数，实际是 %s", t.index.Kind())}
		}
		if i.Value < 0 || i.Value >= int64(len(target.Elements)) {
			return nil, &EvalError{Pos: t.pos, Reason: fmt.Sprintf("数组下标越界：长度 %d，下标 %d", len(target.Elements), i.Value)}
		}
		return target.Elements[i.Value], nil
	case *object.Dict:
		if !object.ValidDictKey(t.index) {
			return nil, &EvalError{Pos: t.pos, Reason: fmt.Sprintf("字典键必须是整数/小数/字符串/布尔，实际是 %s", t.index.Kind())}
		}
		if val, ok := target.Get(t.index); ok {
			return val, nil
		}
		return object.Null, nil
	}
	return nil, &EvalError{Pos: t.pos, Reason: fmt.Sprintf("%s 不支持下标访问", t.target.Kind())}
}

func (t *indexTarget) set(val object.Object) error {
	switch target := t.target.(type) {
	case *object.Array:
		i, ok := t.index.(*object.Integer)
		if !ok {
			return &EvalError{Pos: t.pos, Reason: fmt.Sprintf("下标必须是整数，实际是 %s", t.index.Kind())}
		}
		if i.Value < 0 || i.Value >= int64(len(target.Elements)) {
			return &EvalError{Pos: t.pos, Reason: fmt.Sprintf("数组下标越界：长度 %d，下标 %d", len(target.Elements), i.Value)}
		}
		target.Elements[i.Value] = val
		return nil
	case *object.Dict:
		if !object.ValidDictKey(t.index) {
			return &EvalError{Pos: t.pos, Reason: fmt.Sprintf("字典键必须是整数/小数/字符串/布尔，实际是 %s", t.index.Kind())}
		}
		target.Set(t.index, val)
		return nil
	}
	return &EvalError{Pos: t.pos, Reason: fmt.Sprintf("%s 不支持下标赋值", t.target.Kind())}
}

type fieldTarget struct {
	target object.Object
	name   string
	pos    ast.Position
	env    *Environment
}

func (t *fieldTarget) checkFieldAccess(inst *Instance) error {
	r := inst.resolveField(t.name)
	if !r.found {
		return nil
	}
	if r.ambiguous {
		return &EvalError{Pos: t.pos, Reason: fmt.Sprintf("%s 的字段 %s 在嵌入链中有歧义，需通过嵌入名限定访问", inst.Class.Name, t.name)}
	}
	vis := ownerFieldVis(r.owner, t.name)
	if !memberAccessible(vis, r.owner, t.env, r.owner != inst.Class) {
		return &EvalError{Pos: t.pos, Reason: fmt.Sprintf("%s 的私有字段 %s 不可在模型外访问", r.owner.Name, t.name)}
	}
	return nil
}

func (t *fieldTarget) get() (object.Object, error) {
	switch target := t.target.(type) {
	case *Instance:
		if r := target.resolveField(t.name); r.found {
			if err := t.checkFieldAccess(target); err != nil {
				return nil, err
			}
			return r.value, nil
		}
		return nil, &EvalError{Pos: t.pos, Reason: fmt.Sprintf("%s 没有字段 %s", target.Class.Name, t.name)}
	}
	return nil, &EvalError{Pos: t.pos, Reason: fmt.Sprintf("%s 不支持字段访问", t.target.Kind())}
}

func (t *fieldTarget) set(val object.Object) error {
	switch target := t.target.(type) {
	case *Instance:
		if err := t.checkFieldAccess(target); err != nil {
			return err
		}
		if err := target.setFieldValue(t.name, val); err != nil {
			return &EvalError{Pos: t.pos, Reason: err.Error()}
		}
		return nil
	}
	return &EvalError{Pos: t.pos, Reason: fmt.Sprintf("%s 不支持字段赋值", t.target.Kind())}
}

func resolveAssignTarget(expr ast.Expression, env *Environment) (assignTarget, error) {
	switch node := expr.(type) {
	case *ast.Identifier:
		return &identTarget{name: node.Name, env: env, pos: node.Position}, nil
	case *ast.IndexExpr:
		target, err := Eval(node.Target, env)
		if err != nil {
			return nil, err
		}
		index, err := Eval(node.Index, env)
		if err != nil {
			return nil, err
		}
		return &indexTarget{target: target, index: index, pos: node.Position}, nil
	case *ast.MemberExpr:
		target, err := Eval(node.Object, env)
		if err != nil {
			return nil, err
		}
		return &fieldTarget{target: target, name: node.Name, pos: node.Position, env: env}, nil
	}
	return nil, &EvalError{Pos: expr.Pos(), Reason: "赋值左侧必须是变量、下标或字段"}
}

func evalCompoundAssign(stmt *ast.CompoundAssignStmt, env *Environment) (object.Object, error) {
	target, err := resolveAssignTarget(stmt.Target, env)
	if err != nil {
		return nil, err
	}
	current, err := target.get()
	if err != nil {
		return nil, err
	}
	right, err := Eval(stmt.Value, env)
	if err != nil {
		return nil, err
	}
	result, err := evalBinaryOp(&ast.BinaryExpr{Position: stmt.Position, Op: stmt.Op}, current, right)
	if err != nil {
		return nil, err
	}
	if err := target.set(result); err != nil {
		return nil, err
	}
	return result, nil
}

func evalUpdateExpr(expr *ast.UpdateExpr, env *Environment) (object.Object, error) {
	target, err := resolveAssignTarget(expr.Target, env)
	if err != nil {
		return nil, err
	}
	current, err := target.get()
	if err != nil {
		return nil, err
	}
	one := &object.Integer{Value: 1}
	result, err := evalBinaryOp(&ast.BinaryExpr{Position: expr.Position, Op: expr.Op}, current, one)
	if err != nil {
		return nil, err
	}
	if err := target.set(result); err != nil {
		return nil, err
	}
	if expr.Prefix {
		return result, nil
	}
	return current, nil
}
