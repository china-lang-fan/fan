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
	MethodVis    map[string]ast.Visibility
	Embeds       []*Class
	Parent       *Environment
	TagInstances []object.Object
	Impl         tagsys.Callable
	// SourceFile 是模型定义所在文件的绝对路径，用于文件私有可见性判断。
	SourceFile string
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
	// Embeds 按声明顺序保存被嵌入模型的子实例，可用模型名访问（d.动物）。
	Embeds []*Instance
}

// embedByName 按嵌入模型名查找子实例。
func (i *Instance) embedByName(name string) *Instance {
	for _, e := range i.Embeds {
		if e.Class != nil && e.Class.Name == name {
			return e
		}
	}
	return nil
}

// fieldResolution 描述一次字段提升查找的结果。
type fieldResolution struct {
	value    object.Object
	owner    *Class
	instance *Instance
	// ambiguous 表示同层多个嵌入提供同名字段。
	ambiguous bool
	found     bool
}

// resolveField 沿嵌入链查找字段：自身字段优先；同层多个嵌入提供同名
// 字段时返回 ambiguous；深层唯一提供时返回该字段。
func (i *Instance) resolveField(name string) fieldResolution {
	if v, ok := i.Fields[name]; ok {
		return fieldResolution{value: v, owner: i.Class, instance: i, found: true}
	}
	var matches []fieldResolution
	for _, e := range i.Embeds {
		r := e.resolveField(name)
		if r.found && !r.ambiguous {
			matches = append(matches, r)
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	if len(matches) > 1 {
		return fieldResolution{ambiguous: true, found: true}
	}
	return fieldResolution{}
}

// setFieldValue 沿嵌入链写字段；同层歧义时拒绝写入。
func (i *Instance) setFieldValue(name string, val object.Object) error {
	if _, ok := i.Fields[name]; ok {
		i.Fields[name] = val
		return nil
	}
	var targets []*Instance
	for _, e := range i.Embeds {
		if r := e.resolveField(name); r.found && !r.ambiguous {
			targets = append(targets, e)
		}
	}
	if len(targets) == 1 {
		return targets[0].setFieldValue(name, val)
	}
	if len(targets) > 1 {
		return fmt.Errorf("%s 的字段 %s 在嵌入链中有歧义，需通过嵌入名限定访问", i.Class.Name, name)
	}
	return fmt.Errorf("%s 没有字段 %s", i.Class.Name, name)
}

// methodResolution 描述方法查找结果。
type methodResolution struct {
	fn        *Function
	vis       ast.Visibility
	owner     *Class
	ambiguous bool
	found     bool
}

// resolveMethod 沿嵌入链查找方法：自身方法优先；同层多个嵌入提供同名
// 方法时歧义。
func (c *Class) resolveMethod(name string) methodResolution {
	if fn, ok := c.Methods[name]; ok {
		return methodResolution{fn: fn, vis: c.MethodVis[name], owner: c, found: true}
	}
	var matches []methodResolution
	for _, embed := range c.Embeds {
		r := embed.resolveMethod(name)
		if r.found && !r.ambiguous {
			matches = append(matches, r)
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	if len(matches) > 1 {
		return methodResolution{ambiguous: true, found: true}
	}
	return methodResolution{}
}

// newZeroInstance 递归创建模型及其嵌入链的零值实例（不调用 初始化）。
func newZeroInstance(cls *Class) *Instance {
	inst := &Instance{
		Class:  cls,
		Fields: map[string]object.Object{},
	}
	for _, field := range cls.Fields {
		inst.Fields[field.Name] = object.Null
	}
	for _, embed := range cls.Embeds {
		inst.Embeds = append(inst.Embeds, newZeroInstance(embed))
	}
	return inst
}

func (i *Instance) Kind() object.Kind { return "实例" }
func (i *Instance) Inspect() string {
	return fmt.Sprintf("<%s 实例>", i.Class.Name)
}

func evalClassStmt(stmt *ast.ClassStmt, env *Environment) (object.Object, error) {
	cls := &Class{
		Name:       stmt.Name,
		Fields:     append([]ast.FieldDecl(nil), stmt.Fields...),
		Methods:    map[string]*Function{},
		MethodVis:  map[string]ast.Visibility{},
		Embeds:     nil,
		Parent:     env,
		SourceFile: env.SourceFile,
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
	if err := processClassTags(env.Runtime(), cls, stmt.Tags, env); err != nil {
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
	if stmt.Primitive {
		return evalPrimitiveMethodDef(stmt, env)
	}
	val, ok := env.get(stmt.ClassName)
	if !ok {
		return nil, &EvalError{Pos: stmt.Position, Reason: fmt.Sprintf("模型 %s 未定义", stmt.ClassName)}
	}
	switch owner := val.(type) {
	case *Class:
		return evalClassMethodDef(stmt, env, owner)
	case *NamedType:
		fn := &Function{
			Params:      append([]ast.Parameter(nil), stmt.Function.Params...),
			Body:        stmt.Function.Body,
			Env:         env,
			Name:        stmt.MethodName,
			ReturnTypes: stmt.Function.ReturnTypes,
		}
		if err := processFunctionTags(env.Runtime(), fn, stmt.Function.Tags, env); err != nil {
			return nil, err
		}
		owner.Methods[stmt.MethodName] = fn
		return object.Null, nil
	}
	return nil, &EvalError{Pos: stmt.Position, Reason: fmt.Sprintf("%s 不是模型或自定义类型", stmt.ClassName)}
}

func evalClassMethodDef(stmt *ast.MethodDef, env *Environment, cls *Class) (object.Object, error) {
	fn := &Function{
		Params:      append([]ast.Parameter(nil), stmt.Function.Params...),
		Body:        stmt.Function.Body,
		Env:         env,
		Name:        stmt.MethodName,
		ReturnTypes: stmt.Function.ReturnTypes,
		Owner:       cls,
	}
	if err := processFunctionTags(env.Runtime(), fn, stmt.Function.Tags, env); err != nil {
		return nil, err
	}
	cls.Methods[stmt.MethodName] = fn
	cls.MethodVis[stmt.MethodName] = stmt.Visibility
	return object.Null, nil
}

func (c *Class) instantiate(rt *Runtime, pos ast.Position, args []object.Object) (*Instance, error) {
	if rt == nil && c.Parent != nil {
		rt = c.Parent.Runtime()
	}
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
	inst := newZeroInstance(c)
	initFn := c.lookupMethod("初始化")
	if initFn == nil {
		if len(args) != 0 {
			return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("模型 %s 没有 初始化 方法，不应传参", c.Name)}
		}
		return inst, nil
	}
	bound := initFn.bind(inst)
	if _, err := applyFunctionWithPanic(rt, bound, args, pos, nil); err != nil {
		return nil, err
	}
	return inst, nil
}

func errorStackToArray(frames []object.StackFrame) *object.Array {
	elements := make([]object.Object, 0, len(frames))
	for _, f := range frames {
		d := object.NewDict()
		d.Set(&object.String{Value: "函数"}, &object.String{Value: f.Function})
		d.Set(&object.String{Value: "文件"}, &object.String{Value: f.File})
		d.Set(&object.String{Value: "行"}, &object.Integer{Value: int64(f.Line)})
		d.Set(&object.String{Value: "列"}, &object.Integer{Value: int64(f.Column)})
		elements = append(elements, d)
	}
	return &object.Array{Elements: elements}
}

func (c *Class) lookupMethod(name string) *Function {
	r := c.resolveMethod(name)
	if r.ambiguous {
		return nil
	}
	return r.fn
}

// accessibleFrom 判断成员能否从调用环境访问。
// private 成员仅本模型自己的方法可访问（嵌入后也不放开）；
// public 任意可访问；file（默认）需同文件，但经嵌入链提升访问时放开。
// viaEmbed 表示本次访问来自外层模型对嵌入成员的提升访问。
func memberAccessible(vis ast.Visibility, definingClass *Class, env *Environment, viaEmbed bool) bool {
	switch vis {
	case ast.VisPublic:
		return true
	case ast.VisPrivate:
		if fn := currentFunction(env); fn != nil && fn.Owner == definingClass {
			return true
		}
		return false
	default:
		if viaEmbed {
			return true
		}
		if definingClass == nil {
			return true
		}
		return definingClass.SourceFile == "" || env.SourceFile == "" || definingClass.SourceFile == env.SourceFile
	}
}

func currentFunction(env *Environment) *Function {
	if env == nil {
		return nil
	}
	if env.currentFn != nil {
		return env.currentFn
	}
	if env.outer != nil {
		return currentFunction(env.outer)
	}
	return nil
}

// ownerFieldVis 返回字段在定义模型上的可见性。
func ownerFieldVis(cls *Class, name string) ast.Visibility {
	if cls == nil {
		return ast.VisFile
	}
	for _, f := range cls.Fields {
		if f.Name == name {
			return f.Visibility
		}
	}
	return ast.VisFile
}

func (i *Instance) getMethod(name string, env *Environment) (*Function, error) {
	mr := i.Class.resolveMethod(name)
	if !mr.found {
		return nil, fmt.Errorf("%s 没有方法 %s", i.Class.Name, name)
	}
	if mr.ambiguous {
		return nil, fmt.Errorf("%s 的方法 %s 在嵌入链中有歧义，需通过嵌入名限定", i.Class.Name, name)
	}
	if env != nil {
		if !memberAccessible(mr.vis, mr.owner, env, mr.owner != i.Class) {
			return nil, fmt.Errorf("%s 的私有方法 %s 不可在模型外访问", mr.owner.Name, name)
		}
	}
	// 提升自嵌入的方法绑定到子实例。
	receiver := i
	if mr.owner != i.Class {
		if r := i.resolveFieldOwner(mr.owner); r != nil {
			receiver = r
		}
	}
	return mr.fn.bind(receiver), nil
}

// resolveFieldOwner 在嵌入链中找到属于指定模型的子实例。
func (i *Instance) resolveFieldOwner(cls *Class) *Instance {
	if i.Class == cls {
		return i
	}
	for _, e := range i.Embeds {
		if r := e.resolveFieldOwner(cls); r != nil {
			return r
		}
	}
	return nil
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
			return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("模块 %s 没有公开成员 %s", target.Name, node.Name)}
		}
		return v, nil
	case *Instance:
		// 嵌入子实例可通过嵌入模型名访问：d.动物
		if embed := target.embedByName(node.Name); embed != nil {
			return embed, nil
		}
		// 字段（含嵌入提升）
		if r := target.resolveField(node.Name); r.found {
			if r.ambiguous {
				return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 的字段 %s 在嵌入链中有歧义，请用 嵌入名.%s 限定", target.Class.Name, node.Name, node.Name)}
			}
			if !memberAccessible(ownerFieldVis(r.owner, node.Name), r.owner, env, r.owner != target.Class) {
				return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 的私有字段 %s 不可在模型外访问", r.owner.Name, node.Name)}
			}
			return r.value, nil
		}
		// 方法（含嵌入提升）
		if mr := target.Class.resolveMethod(node.Name); mr.found {
			if mr.ambiguous {
				return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 的方法 %s 在嵌入链中有歧义，请用 嵌入名.%s 限定", target.Class.Name, node.Name, node.Name)}
			}
			if !memberAccessible(mr.vis, mr.owner, env, mr.owner != target.Class) {
				return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 的私有方法 %s 不可在模型外访问", mr.owner.Name, node.Name)}
			}
			receiver := target
			if mr.owner != target.Class {
				if r := target.resolveFieldOwner(mr.owner); r != nil {
					receiver = r
				}
			}
			return mr.fn.bind(receiver), nil
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
		case "错误码", "码":
			return &object.String{Value: target.Code}, nil
		case "详情":
			if target.Details != nil {
				return target.Details, nil
			}
			return object.NewDict(), nil
		case "原因":
			if target.Cause != nil {
				return target.Cause, nil
			}
			return object.Null, nil
		case "调用栈":
			return errorStackToArray(target.Stack), nil
		case "追踪":
			return &object.String{Value: target.Trace()}, nil
		}
		return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("错误 没有成员 %s", node.Name)}
	case *object.Named:
		if namedType, ok := env.Runtime().namedTypes[target.TypeName]; ok {
			if fn := namedType.lookupMethod(node.Name); fn != nil {
				return fn, nil
			}
		}
		return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 没有成员 %s", target.TypeName, node.Name)}
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
