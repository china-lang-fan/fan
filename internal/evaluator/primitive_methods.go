package evaluator

import (
	"fmt"

	"fan/internal/ast"
	"fan/internal/object"
)

type primitiveMethodRegistry struct {
	methods map[string]map[string]*Function
}

func newPrimitiveMethodRegistry() *primitiveMethodRegistry {
	return &primitiveMethodRegistry{methods: map[string]map[string]*Function{}}
}

func primitiveKindFromName(name string) (string, bool) {
	switch name {
	case "字符串":
		return "string", true
	case "整数":
		return "integer", true
	case "小数":
		return "float", true
	case "布尔":
		return "boolean", true
	case "数组":
		return "array", true
	case "字典":
		return "dict", true
	}
	return "", false
}

func (r *primitiveMethodRegistry) register(kind string, name string, fn *Function) {
	if r.methods[kind] == nil {
		r.methods[kind] = map[string]*Function{}
	}
	r.methods[kind][name] = fn
}

func (r *primitiveMethodRegistry) lookup(value object.Object, name string) (*Function, bool) {
	methods, ok := r.methods[runtimeTypeName(value)]
	if !ok {
		return nil, false
	}
	fn, ok := methods[name]
	return fn, ok
}

func (e *Environment) primitiveMethods() *primitiveMethodRegistry {
	if e.primitiveRegistry != nil {
		return e.primitiveRegistry
	}
	if e.outer != nil {
		return e.outer.primitiveMethods()
	}
	e.primitiveRegistry = newPrimitiveMethodRegistry()
	return e.primitiveRegistry
}

func evalPrimitiveMethodDef(stmt *ast.MethodDef, env *Environment) (object.Object, error) {
	kind, ok := primitiveKindFromName(stmt.ClassName)
	if !ok {
		return nil, &EvalError{Pos: stmt.Position, Reason: fmt.Sprintf("不支持的原生类型 %s", stmt.ClassName)}
	}
	if kind != "string" {
		return nil, &EvalError{Pos: stmt.Position, Reason: "原生类型方法当前仅支持字符串"}
	}
	receiverType := ast.TypeAny
	switch kind {
	case "string":
		receiverType = ast.TypeString
	case "integer":
		receiverType = ast.TypeInt
	case "float":
		receiverType = ast.TypeFloat
	case "boolean":
		receiverType = ast.TypeBool
	case "array":
		receiverType = ast.TypeArray
	case "dict":
		receiverType = ast.TypeDict
	}
	params := make([]ast.Parameter, 0, len(stmt.Function.Params)+1)
	params = append(params, ast.Parameter{Name: stmt.ReceiverName, Type: receiverType})
	params = append(params, stmt.Function.Params...)
	fn := &Function{
		Params:      params,
		Body:        stmt.Function.Body,
		Env:         env,
		Name:        stmt.MethodName,
		ReturnTypes: stmt.Function.ReturnTypes,
	}
	if err := processFunctionTags(fn, stmt.Function.Tags, env); err != nil {
		return nil, err
	}
	env.primitiveMethods().register(kind, stmt.MethodName, fn)
	return object.Null, nil
}
