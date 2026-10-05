package evaluator

import (
	"fmt"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/tagsys"
)

type NativeFunction func(args []object.Object) ([]object.Object, error)

type nativeCallable struct {
	name string
	fn   NativeFunction
}

func (c *nativeCallable) Kind() object.Kind { return "函数" }
func (c *nativeCallable) Inspect() string {
	return fmt.Sprintf("<函数 %s>", c.name)
}

func (c *nativeCallable) Call(args []tagsys.Object) ([]tagsys.Object, error) {
	callArgs := make([]object.Object, len(args))
	for i, arg := range args {
		value, ok := arg.(object.Object)
		if !ok || arg == nil {
			value = object.Null
		}
		callArgs[i] = value
	}
	values, err := c.fn(callArgs)
	if err != nil {
		return nil, err
	}
	out := make([]tagsys.Object, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out, nil
}

type classCallable struct {
	cls *Class
}

func (c *classCallable) Kind() object.Kind { return "函数" }
func (c *classCallable) Inspect() string {
	return fmt.Sprintf("<函数 %s>", c.cls.Name)
}

func (c *classCallable) Call(args []tagsys.Object) ([]tagsys.Object, error) {
	callArgs := make([]object.Object, len(args))
	for i, arg := range args {
		callArgs[i] = arg.(object.Object)
	}
	inst, err := c.cls.instantiate(ast.Position{}, callArgs)
	if err != nil {
		return nil, err
	}
	return []tagsys.Object{inst}, nil
}

type variadicUnwrapCallable struct {
	callable tagsys.Callable
}

func (c *variadicUnwrapCallable) Kind() object.Kind { return "函数" }
func (c *variadicUnwrapCallable) Inspect() string {
	return "<函数 可变参数>"
}

func (c *variadicUnwrapCallable) Call(args []tagsys.Object) ([]tagsys.Object, error) {
	callArgs := make([]tagsys.Object, len(args))
	copy(callArgs, args)
	if len(callArgs) > 0 {
		if elements, ok := callArgs[len(callArgs)-1].(*object.Array); ok {
			callArgs = callArgs[:len(callArgs)-1]
			for _, value := range elements.Elements {
				callArgs = append(callArgs, value)
			}
		}
	}
	return c.callable.Call(callArgs)
}

type methodWithoutReceiverCallable struct {
	callable tagsys.Callable
}

func (c *methodWithoutReceiverCallable) Kind() object.Kind { return "函数" }
func (c *methodWithoutReceiverCallable) Inspect() string {
	return "<函数 无接收者>"
}

func (c *methodWithoutReceiverCallable) Call(args []tagsys.Object) ([]tagsys.Object, error) {
	if len(args) == 0 {
		return c.callable.Call(args)
	}
	return c.callable.Call(args[1:])
}

type fanCallable struct {
	fn *Function
}

func (c *fanCallable) Kind() object.Kind { return "函数" }
func (c *fanCallable) Inspect() string {
	return fmt.Sprintf("<函数 %s>", c.fn.Name)
}

func (c *fanCallable) Call(args []tagsys.Object) ([]tagsys.Object, error) {
	return c.fn.Call(args)
}

type runtimeTagContext struct {
	tag      object.Object
	target   tagsys.Target
	callable tagsys.Callable
}

func (c *runtimeTagContext) Tag() tagsys.Object { return c.tag }
func (c *runtimeTagContext) Target() tagsys.Target {
	return c.target
}
func (c *runtimeTagContext) Callable() tagsys.Callable {
	return c.callable
}
func (c *runtimeTagContext) ReplaceCallable(callable tagsys.Callable) {
	switch target := c.target.(type) {
	case *Function:
		if target.Owner == nil {
			target.Impl = callable
			return
		}
		target.MethodImpl = callable
	case *Class:
		target.Impl = callable
	}
}

func (c *runtimeTagContext) ReplaceFunction(callable tagsys.Callable) {
	if fn, ok := c.target.(*Function); ok && fn.Owner == nil {
		fn.Impl = callable
	}
}

func (c *runtimeTagContext) ReplaceMethod(callable tagsys.Callable) {
	fn, ok := c.target.(*Function)
	if !ok || fn.Owner == nil {
		return
	}
	fn.MethodImpl = callable
}

func (c *runtimeTagContext) member(name string) (object.Object, error) {
	switch name {
	case "标签", "Tag":
		return c.tag, nil
	case "目标":
		return c.target.(object.Object), nil
	case "当前实现":
		return c.callable.(object.Object), nil
	case "替换实现":
		return &tagContextReplaceFn{ctx: c}, nil
	}
	return nil, fmt.Errorf("标签上下文没有成员 %s", name)
}

type tagContextReplaceFn struct {
	ctx *runtimeTagContext
}

func (f *tagContextReplaceFn) Kind() object.Kind { return "函数" }
func (f *tagContextReplaceFn) Inspect() string {
	return "<函数 替换实现>"
}

func (f *tagContextReplaceFn) Call(args []tagsys.Object) ([]tagsys.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("替换实现 需要 1 个参数")
	}
	var callable tagsys.Callable
	switch value := args[0].(type) {
	case tagsys.Callable:
		callable = value
	case *Function:
		callable = value
	default:
		return nil, fmt.Errorf("替换实现 的参数必须是可调用对象")
	}
	f.ctx.ReplaceCallable(callable)
	return nil, nil
}

func (c *runtimeTagContext) Kind() object.Kind { return "标签上下文" }
func (c *runtimeTagContext) Inspect() string {
	return "<标签上下文>"
}

func (c *runtimeTagContext) 标签() object.Object { return c.tag }
func (c *runtimeTagContext) 目标() tagsys.Target {
	return c.target
}
func (c *runtimeTagContext) 当前实现() tagsys.Callable {
	return c.callable
}
func (c *runtimeTagContext) 替换实现(callable tagsys.Callable) {
	c.ReplaceCallable(callable)
}

var builtinClasses = map[string]*Class{}
var nativeFunctions = map[string]tagsys.Callable{}
var goTagHandlers = map[string]tagsys.Handler{}
var fanTagHandlers = map[string]*Function{}

func RegisterBuiltinClass(name string, fields []ast.FieldDecl) *Class {
	cls := &Class{
		Name:    name,
		Fields:  append([]ast.FieldDecl(nil), fields...),
		Methods: map[string]*Function{},
	}
	builtinClasses[name] = cls
	return cls
}

func RegisterNativeFunction(name string, fn NativeFunction) {
	nativeFunctions[name] = &nativeCallable{name: name, fn: fn}
}

func RegisterTagHandler(name string, handler tagsys.Handler) {
	goTagHandlers[name] = handler
}

func registerFanTagHandler(name string, fn *Function) {
	fanTagHandlers[name] = fn
}

func init() {
	RegisterBuiltinClass("内建", []ast.FieldDecl{
		{Name: "值", Type: ast.TypeString},
	})
	RegisterBuiltinClass("测试", nil)
	RegisterBuiltinClass("test", nil)
	RegisterTagHandler("内建", tagsys.HandlerFunc(builtinTagHandler))
	RegisterNativeFunction("print", nativePrint)
	RegisterNativeFunction("打印", nativePrint)
	RegisterNativeFunction("测试.加一", func(args []object.Object) ([]object.Object, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("测试.加一 需要 1 个参数")
		}
		n, ok := args[0].(*object.Integer)
		if !ok {
			return nil, fmt.Errorf("测试.加一 参数必须是整数")
		}
		return []object.Object{&object.Integer{Value: n.Value + 1}}, nil
	})
	RegisterNativeFunction("测试.基础实例", func(args []object.Object) ([]object.Object, error) {
		cls := &Class{
			Name:    "基础",
			Fields:  []ast.FieldDecl{{Name: "值", Type: ast.TypeString}},
			Methods: map[string]*Function{},
		}
		return []object.Object{&Instance{Class: cls, Fields: map[string]object.Object{"值": &object.String{Value: "构造替换"}}}}, nil
	})
}

func nativePrint(args []object.Object) ([]object.Object, error) {
	values := args
	if len(args) == 1 {
		if elements, ok := args[0].(*object.Array); ok {
			values = elements.Elements
		}
	}
	printObjects(values)
	return []object.Object{object.Null}, nil
}

func builtinTagHandler(ctx tagsys.Context) error {
	tag, ok := ctx.Tag().(*Instance)
	if !ok || tag.Class.Name != "内建" {
		return fmt.Errorf("内建标签格式错误")
	}
	value, exists := tag.getField("值")
	if !exists {
		return fmt.Errorf("内建标签缺少值")
	}
	name, ok := value.(*object.String)
	if !ok {
		return fmt.Errorf("内建标签值必须是字符串")
	}
	fn, ok := nativeFunctions[name.Value]
	if !ok {
		return fmt.Errorf("内建函数未注册：%s", name.Value)
	}
	if targetIsVariadic(ctx.Target()) {
		fn = &variadicUnwrapCallable{callable: fn}
	}
	if ctx.Target().TargetKind() == tagsys.TargetMethod {
		ctx.ReplaceMethod(&methodWithoutReceiverCallable{callable: fn})
		return nil
	}
	ctx.ReplaceCallable(fn)
	return nil
}

func resolveClass(name string, env *Environment) (*Class, bool) {
	if cls, ok := builtinClasses[name]; ok {
		return cls, true
	}
	value, ok := env.get(name)
	if !ok {
		return nil, false
	}
	cls, ok := value.(*Class)
	return cls, ok
}

func createTagInstance(cls *Class, tagNode *ast.TagExpr, env *Environment) (*Instance, error) {
	if len(tagNode.Positional) > 1 {
		return nil, fmt.Errorf("标签未命名参数最多一个")
	}
	inst := &Instance{Class: cls, Fields: map[string]object.Object{}}
	for _, field := range cls.Fields {
		inst.Fields[field.Name] = object.Null
	}
	for _, embed := range cls.Embeds {
		for _, field := range embed.Fields {
			if _, exists := inst.Fields[field.Name]; !exists {
				inst.Fields[field.Name] = object.Null
			}
		}
	}
	if len(tagNode.Positional) == 1 {
		value, err := Eval(tagNode.Positional[0], env)
		if err != nil {
			return nil, err
		}
		if value == nil {
			value = object.Null
		}
		if _, exists := inst.Fields["值"]; !exists {
			return nil, fmt.Errorf("标签模型 %s 没有字段 值", cls.Name)
		}
		inst.Fields["值"] = value
	}
	for _, arg := range tagNode.Named {
		value, err := Eval(arg.Value, env)
		if err != nil {
			return nil, err
		}
		if value == nil {
			value = object.Null
		}
		if _, exists := inst.Fields[arg.Name]; !exists {
			return nil, fmt.Errorf("标签模型 %s 没有字段 %s", cls.Name, arg.Name)
		}
		inst.Fields[arg.Name] = value
	}
	return inst, nil
}

func processTags(target tagsys.Target, tagExprs []*ast.TagExpr, env *Environment) error {
	for _, tagNode := range tagExprs {
		cls, ok := resolveClass(tagNode.TypeName, env)
		if !ok {
			return &EvalError{Pos: tagNode.Position, Reason: fmt.Sprintf("标签模型 %s 未定义", tagNode.TypeName)}
		}
		instance, err := createTagInstance(cls, tagNode, env)
		if err != nil {
			return &EvalError{Pos: tagNode.Position, Reason: err.Error()}
		}
		attachTag(target, instance)
		callable := initialCallable(target)
		ctx := &runtimeTagContext{tag: instance, target: target, callable: callable}
		if handler, ok := goTagHandlers[tagNode.TypeName]; ok {
			if err := handler.Handle(ctx); err != nil {
				return &EvalError{Pos: tagNode.Position, Reason: err.Error()}
			}
			continue
		}
		if handler, ok := fanTagHandlers[tagNode.TypeName]; ok {
			if _, err := handler.Call([]tagsys.Object{ctx}); err != nil {
				return &EvalError{Pos: tagNode.Position, Reason: err.Error()}
			}
		}
	}
	return nil
}

func attachTag(target tagsys.Target, tag *Instance) {
	switch value := target.(type) {
	case *Function:
		value.TagInstances = append(value.TagInstances, tag)
	case *Class:
		value.TagInstances = append(value.TagInstances, tag)
	}
}

func initialCallable(target tagsys.Target) tagsys.Callable {
	switch value := target.(type) {
	case *Function:
		if value.Impl != nil {
			return value.Impl
		}
		if value.MethodImpl != nil {
			return value.MethodImpl
		}
		return &fanCallable{fn: value}
	case *Class:
		if value.Impl != nil {
			return value.Impl
		}
		return &classCallable{cls: value}
	}
	return nil
}

func callTagsysCallable(callable tagsys.Callable, args []object.Object, pos ast.Position) (object.Object, error) {
	callArgs := make([]tagsys.Object, len(args))
	for i, arg := range args {
		callArgs[i] = arg
	}
	results, err := callable.Call(callArgs)
	if err != nil {
		return nil, err
	}
	if len(results) == 1 {
		if result, ok := results[0].(object.Object); ok {
			return result, nil
		}
	}
	return object.Null, nil
}

func callDynamicMember(obj object.Object, member *ast.MemberExpr, args []object.Object, pos ast.Position) (object.Object, error) {
	field, err := evalFieldAccess(member, nil, obj)
	if err != nil {
		return nil, err
	}
	switch callable := field.(type) {
	case *Function:
		return applyFunction(callable, args, pos)
	case tagsys.Callable:
		return callTagsysCallable(callable, args, pos)
	}
	return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("%s 不是可调用对象", member.Name)}
}

func processFunctionTags(fn *Function, tags []*ast.TagExpr, env *Environment) error {
	return processTags(fn, tags, env)
}

func processClassTags(cls *Class, tags []*ast.TagExpr, env *Environment) error {
	return processTags(cls, tags, env)
}
