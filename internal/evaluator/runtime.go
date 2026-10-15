package evaluator

import (
	"io"
	"os"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/tagsys"
)

// Runtime 是一次独立的凡程序运行实例，承载此前散落在包级变量中的
// 全部可变状态：调用帧（稍后/抛出/捕获）、标准输出、进程上下文、
// 标签处理器、内建类与原生函数表。
//
// 不同 Runtime 之间完全隔离，可以在同一个进程中并发或重入地执行
// 多个凡程序。内置原生函数通过 init() 注册到进程级只读模板，
// NewRuntime 时复制一份，凡代码通过“注册标签处理”所做的修改
// 只影响当前 Runtime。
type Runtime struct {
	frames  []*callFrame
	fnStack []fnFrame

	stdout io.Writer

	scriptPath string
	args       []string

	builtinClasses  map[string]*Class
	interfaces      map[string]*Interface
	nativeFunctions map[string]tagsys.Callable
	goTagHandlers   map[string]tagsys.Handler
	fanTagHandlers  map[string]*Function
}

// fnFrame 记录一次凡函数调用，用于错误调用栈。
type fnFrame struct {
	name string
	pos  ast.Position
}

// NewRuntime 创建一个独立运行实例。
func NewRuntime() *Runtime {
	r := &Runtime{
		stdout:         os.Stdout,
		args:           []string{},
		builtinClasses: cloneBuiltinClasses(),
		interfaces:     map[string]*Interface{},
		goTagHandlers:  cloneGoTagHandlers(),
		fanTagHandlers: map[string]*Function{},
	}
	r.nativeFunctions = cloneNativeFunctions(r)
	return r
}

// SetStdout 设置打印输出目标。
func (r *Runtime) SetStdout(w io.Writer) {
	if w != nil {
		r.stdout = w
	}
}

// Stdout 返回当前打印输出目标。
func (r *Runtime) Stdout() io.Writer {
	if r.stdout == nil {
		return os.Stdout
	}
	return r.stdout
}

// SetProcessContext 设置脚本路径与命令行参数。
func (r *Runtime) SetProcessContext(scriptPath string, args []string) {
	copied := make([]string, len(args))
	copy(copied, args)
	r.scriptPath = scriptPath
	r.args = copied
}

func (r *Runtime) processContext() processContext {
	args := make([]string, len(r.args))
	copy(args, r.args)
	return processContext{scriptPath: r.scriptPath, args: args}
}

func (r *Runtime) pushFn(name string, pos ast.Position) {
	r.fnStack = append(r.fnStack, fnFrame{name: name, pos: pos})
}

func (r *Runtime) popFn() {
	if len(r.fnStack) > 0 {
		r.fnStack = r.fnStack[:len(r.fnStack)-1]
	}
}

// Stack 复制当前调用栈（最内层在前）。
func (r *Runtime) Stack() []object.StackFrame {
	out := make([]object.StackFrame, 0, len(r.fnStack))
	for i := len(r.fnStack) - 1; i >= 0; i-- {
		f := r.fnStack[i]
		out = append(out, object.StackFrame{
			Function: f.name,
			File:     r.scriptPath,
			Line:     f.pos.Line,
			Column:   f.pos.Column,
		})
	}
	return out
}

// --- 调用帧：稍后 / 抛出 / 捕获 ---

func (r *Runtime) pushFrame() *callFrame {
	frame := &callFrame{}
	r.frames = append(r.frames, frame)
	return frame
}

func (r *Runtime) popFrame() {
	if len(r.frames) > 0 {
		r.frames = r.frames[:len(r.frames)-1]
	}
}

func (r *Runtime) currentFrame() *callFrame {
	if len(r.frames) == 0 {
		return nil
	}
	return r.frames[len(r.frames)-1]
}

// --- 标签与原生函数表 ---

func (r *Runtime) registerBuiltinClass(name string, fields []ast.FieldDecl) *Class {
	cls := &Class{
		Name:    name,
		Fields:  append([]ast.FieldDecl(nil), fields...),
		Methods: map[string]*Function{},
	}
	r.builtinClasses[name] = cls
	return cls
}

func (r *Runtime) registerNativeFunction(name string, fn NativeFunction) {
	r.nativeFunctions[name] = &nativeCallable{name: name, rt: r, fn: fn}
}

func (r *Runtime) registerTagHandler(name string, handler tagsys.Handler) {
	r.goTagHandlers[name] = handler
}

func (r *Runtime) registerFanTagHandler(name string, fn *Function) {
	r.fanTagHandlers[name] = fn
}

func (r *Runtime) resolveClass(name string, env *Environment) (*Class, bool) {
	if cls, ok := r.builtinClasses[name]; ok {
		return cls, true
	}
	value, ok := env.get(name)
	if !ok {
		return nil, false
	}
	cls, ok := value.(*Class)
	return cls, ok
}

func (r *Runtime) printObjects(args []object.Object) {
	values := args
	if len(args) == 1 {
		if elements, ok := args[0].(*object.Array); ok {
			values = elements.Elements
		}
	}
	writeObjects(r.Stdout(), values)
}

// NativePrint 供内部与原生模块使用的打印入口，输出到本实例。
func (r *Runtime) NativePrint(args []object.Object) {
	r.printObjects(args)
}
