package evaluator

import (
	"fmt"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/tagsys"
)

const KindFunction object.Kind = "函数"

type Function struct {
	Params       []ast.Parameter
	Body         *ast.BlockStmt
	Env          *Environment
	Name         string
	ReturnTypes  []ast.DeclType
	NameSegments []string
	Connectors   []string
	GroupSizes   []int
	Signature    string
	TagInstances []object.Object
	Impl         tagsys.Callable
	MethodImpl   tagsys.Callable
	Owner        *Class
	Self         *Instance
}

func (f *Function) Kind() object.Kind { return KindFunction }
func (f *Function) Inspect() string {
	if f.Name != "" {
		return fmt.Sprintf("<函数 %s>", f.Name)
	}
	return "<函数>"
}

func (f *Function) TargetName() string { return f.Name }
func (f *Function) TargetKind() tagsys.TargetKind {
	if f.Owner != nil {
		return tagsys.TargetMethod
	}
	return tagsys.TargetFunction
}
func (f *Function) Tags() []tagsys.Object {
	tags := make([]tagsys.Object, len(f.TagInstances))
	for i, tag := range f.TagInstances {
		tags[i] = tag
	}
	return tags
}

type returnSignal struct {
	values []object.Object
}

func (r *returnSignal) Error() string {
	return "return signal"
}

func evalFunctionLiteral(node *ast.FunctionLiteral, env *Environment) (object.Object, error) {
	fn := &Function{
		Params:       node.Params,
		Body:         node.Body,
		Env:          env,
		Name:         node.Name,
		ReturnTypes:  node.ReturnTypes,
		NameSegments: node.NameSegments,
		Connectors:   node.Connectors,
		GroupSizes:   node.GroupSizes,
		Signature:    node.Signature,
	}
	if err := processFunctionTags(fn, node.Tags, env); err != nil {
		return nil, err
	}
	return fn, nil
}

func evalMemberExpression(node *ast.MemberExpr, env *Environment) (object.Object, error) {
	obj, err := Eval(node.Object, env)
	if err != nil {
		return nil, err
	}
	return evalFieldAccess(node, env, obj)
}

func bindArguments(fn *Function, args []object.Object, pos ast.Position) ([]object.Object, error) {
	values := make([]object.Object, len(fn.Params))
	for i, param := range fn.Params {
		if param.Variadic {
			if i != len(fn.Params)-1 {
				return nil, &EvalError{Pos: pos, Reason: "可变参数必须是最后一个参数"}
			}
			if len(args) < i {
				return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("函数参数数量不符：至少需要 %d 个，实际 %d 个", i, len(args))}
			}
			extra := args[i:]
			for _, val := range extra {
				if err := checkDeclType(param.Type, val); err != nil {
					return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("参数 %s 类型不匹配：%s", param.Name, err.Error())}
				}
			}
			values[i] = &object.Array{Elements: append([]object.Object(nil), extra...)}
			return values, nil
		}
		if len(args) > i {
			values[i] = args[i]
			continue
		}
		if param.Default != nil {
			val, err := Eval(param.Default, fn.Env)
			if err != nil {
				return nil, err
			}
			values[i] = val
			continue
		}
		return nil, &EvalError{
			Pos:    pos,
			Reason: fmt.Sprintf("函数参数数量不符：需要 %d 个，实际 %d 个", len(fn.Params), len(args)),
		}
	}
	if len(args) > len(fn.Params) {
		return nil, &EvalError{
			Pos:    pos,
			Reason: fmt.Sprintf("函数参数数量不符：需要 %d 个，实际 %d 个", len(fn.Params), len(args)),
		}
	}
	return values, nil
}

func (f *Function) Call(args []tagsys.Object) ([]tagsys.Object, error) {
	callArgs := make([]object.Object, len(args))
	for i, arg := range args {
		value, ok := arg.(object.Object)
		if !ok || arg == nil {
			value = object.Null
		}
		callArgs[i] = value
	}
	result, err := applyFunction(f, callArgs, ast.Position{})
	if err != nil {
		return nil, err
	}
	switch values := result.(type) {
	case *object.Tuple:
		out := make([]tagsys.Object, len(values.Values))
		for i, value := range values.Values {
			out[i] = value
		}
		return out, nil
	default:
		return []tagsys.Object{result}, nil
	}
}

func callImpl(impl tagsys.Callable, args []tagsys.Object, pos ast.Position, returnTypes []ast.DeclType) (object.Object, error) {
	results, err := impl.Call(args)
	if err != nil {
		return nil, err
	}
	returnValues := make([]object.Object, len(results))
	for i, result := range results {
		value, ok := result.(object.Object)
		if !ok || result == nil {
			value = object.Null
		}
		returnValues[i] = value
	}
	return coerceReturnValues(pos, returnTypes, returnValues)
}

func applyFunction(fn *Function, args []object.Object, pos ast.Position) (object.Object, error) {
	values, err := bindArguments(fn, args, pos)
	if err != nil {
		return nil, err
	}
	if fn.Impl != nil {
		callArgs := make([]tagsys.Object, len(values))
		for i, value := range values {
			callArgs[i] = value
		}
		return callImpl(fn.Impl, callArgs, pos, fn.ReturnTypes)
	}
	if fn.MethodImpl != nil {
		callArgs := make([]tagsys.Object, 0, len(values)+1)
		self := object.Object(object.Null)
		if fn.Self != nil {
			self = fn.Self
		}
		callArgs = append(callArgs, self)
		for _, value := range values {
			callArgs = append(callArgs, value)
		}
		return callImpl(fn.MethodImpl, callArgs, pos, fn.ReturnTypes)
	}
	env := NewEnclosedEnvironment(fn.Env)
	for i, param := range fn.Params {
		if param.Variadic {
			if err := env.declare(param.Name, values[i], false, ast.TypeArray); err != nil {
				return nil, &EvalError{Pos: pos, Reason: err.Error()}
			}
			continue
		}
		if err := checkDeclType(param.Type, values[i]); err != nil {
			return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("参数 %s 类型不匹配：%s", param.Name, err.Error())}
		}
		if err := env.declare(param.Name, values[i], false, param.Type); err != nil {
			return nil, &EvalError{Pos: pos, Reason: err.Error()}
		}
	}
	result, err := evalBlock(fn.Body, env)
	var resultValues []object.Object
	if err != nil {
		if sig, ok := err.(*returnSignal); ok {
			resultValues = sig.values
		} else if cs, ok := err.(*checkSignal); ok {
			resultValues, err = fn.checkReturn(cs.err)
			if err != nil {
				return nil, err
			}
			return coerceReturnValues(pos, fn.ReturnTypes, resultValues)
		} else {
			return nil, err
		}
	} else {
		if result == nil {
			result = object.Null
		}
		resultValues = []object.Object{result}
	}
	return coerceReturnValues(pos, fn.ReturnTypes, resultValues)
}

func (f *Function) checkReturn(cause *object.Error) ([]object.Object, error) {
	if len(f.ReturnTypes) == 0 {
		return nil, cause
	}
	last := f.ReturnTypes[len(f.ReturnTypes)-1]
	if last != ast.TypeError {
		return nil, cause
	}
	values := make([]object.Object, len(f.ReturnTypes))
	for i, dt := range f.ReturnTypes[:len(f.ReturnTypes)-1] {
		values[i] = zeroValueForType(dt)
	}
	values[len(values)-1] = cause
	return values, nil
}

func coerceReturnValues(pos ast.Position, types []ast.DeclType, values []object.Object) (object.Object, error) {
	if len(types) == 0 {
		if len(values) <= 1 {
			return values[0], nil
		}
		return &object.Tuple{Values: values}, nil
	}
	if len(types) != len(values) {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("返回值数量不符：声明 %d 个，实际 %d 个", len(types), len(values))}
	}
	for i, dt := range types {
		if values[i] == nil {
			values[i] = object.Null
		}
		if dt == ast.TypeError && values[i].Kind() == object.KindNil {
			continue
		}
		if err := checkDeclType(dt, values[i]); err != nil {
			return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("第 %d 个返回值%s", i+1, err.Error())}
		}
	}
	if len(values) == 1 {
		return values[0], nil
	}
	return &object.Tuple{Values: values}, nil
}
