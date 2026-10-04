package evaluator

import (
	"fmt"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/tagsys"
)

const KindClass object.Kind = "类"

type Class struct {
	Name         string
	Fields       []ast.FieldDecl
	Methods      map[string]*Function
	Embeds       []*Class
	Parent       *Environment
	TagInstances []object.Object
	Impl         tagsys.Callable
}

func (c *Class) Kind() object.Kind { return KindClass }
func (c *Class) Inspect() string   { return fmt.Sprintf("<模型 %s>", c.Name) }

func (c *Class) TargetName() string { return c.Name }
func (c *Class) TargetKind() tagsys.TargetKind {
	return tagsys.TargetClass
}
func (c *Class) Tags() []tagsys.Object {
	tags := make([]tagsys.Object, len(c.TagInstances))
	for i, tag := range c.TagInstances {
		tags[i] = tag
	}
	return tags
}

type Instance struct {
	Class  *Class
	Fields map[string]object.Object
}

func (i *Instance) Kind() object.Kind { return "实例" }
func (i *Instance) Inspect() string {
	return fmt.Sprintf("<%s 实例>", i.Class.Name)
}

func evalClassStmt(stmt *ast.ClassStmt, env *Environment) (object.Object, error) {
	cls := &Class{
		Name:    stmt.Name,
		Fields:  append([]ast.FieldDecl(nil), stmt.Fields...),
		Methods: map[string]*Function{},
		Parent:  env,
	}
	for _, embedName := range stmt.Embeds {
		val, ok := env.get(embedName)
		if !ok {
			return nil, &EvalError{Pos: stmt.Position, Reason: fmt.Sprintf("嵌入的模型 %s 未定义", embedName)}
		}
		embedCls, ok := val.(*Class)
		if !ok {
			return nil, &EvalError{Pos: stmt.Position, Reason: fmt.Sprintf("%s 不是模型", embedName)}
		}
		cls.Embeds = append(cls.Embeds, embedCls)
	}
	if err := processClassTags(cls, stmt.Tags, env); err != nil {
		return nil, err
	}
	if existing, ok := env.get(stmt.Name); ok {
		if existingCls, ok := existing.(*Class); ok {
			existingCls.Fields = cls.Fields
			existingCls.Embeds = cls.Embeds
			existingCls.Parent = env
			existingCls.TagInstances = append(existingCls.TagInstances, cls.TagInstances...)
			existingCls.Impl = cls.Impl
			return existingCls, nil
		}
	}
	if err := env.declare(stmt.Name, cls, false, ast.TypeAny); err != nil {
		return nil, &EvalError{Pos: stmt.Position, Reason: err.Error()}
	}
	return cls, nil
}

func evalMethodDef(stmt *ast.MethodDef, env *Environment) (object.Object, error) {
	val, ok := env.get(stmt.ClassName)
	if !ok {
		return nil, &EvalError{Pos: stmt.Position, Reason: fmt.Sprintf("模型 %s 未定义", stmt.ClassName)}
	}
	cls, ok := val.(*Class)
	if !ok {
		return nil, &EvalError{Pos: stmt.Position, Reason: fmt.Sprintf("%s 不是模型", stmt.ClassName)}
	}
	fn := &Function{
		Params:      append([]ast.Parameter(nil), stmt.Function.Params...),
		Body:        stmt.Function.Body,
		Env:         env,
		Name:        stmt.MethodName,
		ReturnTypes: stmt.Function.ReturnTypes,
		Owner:       cls,
	}
	if err := processFunctionTags(fn, stmt.Function.Tags, env); err != nil {
		return nil, err
	}
	cls.Methods[stmt.MethodName] = fn
	return object.Null, nil
}

func (c *Class) instantiate(pos ast.Position, args []object.Object) (*Instance, error) {
	if c.Impl != nil {
		callArgs := make([]tagsys.Object, len(args))
		for i, arg := range args {
			callArgs[i] = arg
		}
		results, err := c.Impl.Call(callArgs)
		if err != nil {
			return nil, err
		}
		if len(results) != 1 {
			return nil, fmt.Errorf("模型 %s 的自定义实现必须返回一个实例", c.Name)
		}
		inst, ok := results[0].(*Instance)
		if !ok {
			return nil, fmt.Errorf("模型 %s 的自定义实现必须返回实例", c.Name)
		}
		return inst, nil
	}
	inst := &Instance{
		Class:  c,
		Fields: map[string]object.Object{},
	}
	for _, field := range c.Fields {
		inst.Fields[field.Name] = object.Null
	}
	for _, embed := range c.Embeds {
		for _, field := range embed.Fields {
			if _, exists := inst.Fields[field.Name]; !exists {
				inst.Fields[field.Name] = object.Null
			}
		}
	}
	initFn := c.lookupMethod("初始化")
	if initFn == nil {
		if len(args) != 0 {
			return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("模型 %s 没有 初始化 方法，不应传参", c.Name)}
		}
		return inst, nil
	}
	bound := initFn.bind(inst)
	if _, err := applyFunction(bound, args, pos); err != nil {
		return nil, err
	}
	return inst, nil
}

func (c *Class) lookupMethod(name string) *Function {
	if fn, ok := c.Methods[name]; ok {
		return fn
	}
	var found []*Function
	for _, embed := range c.Embeds {
		if fn := embed.lookupMethod(name); fn != nil {
			found = append(found, fn)
		}
	}
	if len(found) == 1 {
		return found[0]
	}
	if len(found) > 1 {
		return nil
	}
	return nil
}

func (i *Instance) getField(name string) (object.Object, bool) {
	v, ok := i.Fields[name]
	return v, ok
}

func (i *Instance) setField(name string, val object.Object) error {
	if _, exists := i.Fields[name]; !exists {
		return fmt.Errorf("%s 没有字段 %s", i.Class.Name, name)
	}
	i.Fields[name] = val
	return nil
}

func (i *Instance) getMethod(name string) (*Function, error) {
	fn := i.Class.lookupMethod(name)
	if fn == nil {
		return nil, fmt.Errorf("%s 没有方法 %s", i.Class.Name, name)
	}
	return fn.bind(i), nil
}

func (f *Function) bind(inst *Instance) *Function {
	bound := &Function{
		Params:       append([]ast.Parameter(nil), f.Params...),
		Body:         f.Body,
		Env:          NewEnclosedEnvironment(f.Env),
		Name:         f.Name,
		ReturnTypes:  f.ReturnTypes,
		TagInstances: append([]object.Object(nil), f.TagInstances...),
		Impl:         f.Impl,
		MethodImpl:   f.MethodImpl,
		Owner:        f.Owner,
		Self:         inst,
	}
	_ = bound.Env.declare("自己", inst, false, ast.TypeAny)
	return bound
}

func evalFieldAccess(node *ast.MemberExpr, env *Environment, obj object.Object) (object.Object, error) {
	switch target := obj.(type) {
	case *object.Module:
		v, ok := target.Exports[node.Name]
		if !ok {
			return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("模块 %s 没有导出 %s", target.Name, node.Name)}
		}
		return v, nil
	case *Instance:
		if v, ok := target.getField(node.Name); ok {
			return v, nil
		}
		if fn := target.Class.lookupMethod(node.Name); fn != nil {
			return fn.bind(target), nil
		}
		return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 没有字段或方法 %s", target.Class.Name, node.Name)}
	case *object.Array:
		return evalArrayMember(node, target)
	case *object.Dict:
		return evalDictMember(node, target)
	case *object.Error:
		switch node.Name {
		case "消息":
			return &object.String{Value: target.Message}, nil
		}
		return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("错误 没有成员 %s", node.Name)}
	case *runtimeTagContext:
		return target.member(node.Name)
	case *Function:
		switch node.Name {
		case "标签":
			out := make([]object.Object, len(target.TagInstances))
			copy(out, target.TagInstances)
			return &object.Array{Elements: out}, nil
		}
	}
	return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 没有成员 %s", obj.Kind(), node.Name)}
}

func evalArrayMember(node *ast.MemberExpr, arr *object.Array) (object.Object, error) {
	switch node.Name {
	case "长度":
		return &object.Integer{Value: int64(len(arr.Elements))}, nil
	case "反转":
		out := make([]object.Object, len(arr.Elements))
		for i := range arr.Elements {
			out[i] = arr.Elements[len(arr.Elements)-1-i]
		}
		return &object.Array{Elements: out}, nil
	}
	return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("数组没有成员 %s", node.Name)}
}
