package evaluator

import (
	"fmt"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/tagsys"
)

type panicSignal struct {
	value object.Object
}

func (p *panicSignal) Error() string {
	return object.Format(p.value)
}

type deferredCall struct {
	callable object.Object
	builtin  builtinFn
	args     []object.Object
	pos      ast.Position
}

type callFrame struct {
	defers []deferredCall
	panic  *panicSignal
}

func evalDeferStmt(rt *Runtime, stmt *ast.DeferStmt, env *Environment) (object.Object, error) {
	frame := rt.currentFrame()
	if frame == nil {
		return nil, &EvalError{Pos: stmt.Position, Reason: "稍后 只能在函数或脚本作用域中使用"}
	}
	call, err := prepareDeferredCall(rt, stmt.Call, env, stmt.Position)
	if err != nil {
		return nil, err
	}
	frame.defers = append(frame.defers, call)
	return object.Null, nil
}

func prepareDeferredCall(rt *Runtime, expr ast.Expression, env *Environment, pos ast.Position) (deferredCall, error) {
	switch node := expr.(type) {
	case *ast.FunctionLiteral:
		fn, err := Eval(node, env)
		if err != nil {
			return deferredCall{}, err
		}
		return deferredCall{callable: fn, args: nil, pos: pos}, nil
	case *ast.CallExpr:
		return prepareDeferredValueCall(rt, node, env)
	default:
		return deferredCall{}, &EvalError{Pos: pos, Reason: "稍后 后必须是函数调用或无参函数"}
	}
}

func prepareDeferredValueCall(rt *Runtime, node *ast.CallExpr, env *Environment) (deferredCall, error) {
	var callable object.Object
	switch callee := node.Callee.(type) {
	case *ast.Identifier:
		if fn, ok := builtins[callee.Name]; ok {
			args := make([]object.Object, 0, len(node.Args))
			for _, arg := range node.Args {
				value, err := Eval(arg, env)
				if err != nil {
					return deferredCall{}, err
				}
				if value == nil {
					value = object.Null
				}
				args = append(args, value)
			}
			return deferredCall{builtin: fn, args: flattenSpreadMarkers(spreadTuples(args)), pos: node.Position}, nil
		}
		value, ok := env.get(callee.Name)
		if !ok {
			return deferredCall{}, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("未知函数：%s", callee.Name)}
		}
		callable = value
	case *ast.MemberExpr:
		value, err := Eval(callee, env)
		if err != nil {
			return deferredCall{}, err
		}
		callable = value
	default:
		value, err := Eval(node.Callee, env)
		if err != nil {
			return deferredCall{}, err
		}
		callable = value
	}
	args := make([]object.Object, 0, len(node.Args))
	for _, arg := range node.Args {
		value, err := Eval(arg, env)
		if err != nil {
			return deferredCall{}, err
		}
		if value == nil {
			value = object.Null
		}
		args = append(args, value)
	}
	return deferredCall{callable: callable, args: flattenSpreadMarkers(spreadTuples(args)), pos: node.Position}, nil
}

func evalThrowStmt(stmt *ast.ThrowStmt, env *Environment) (object.Object, error) {
	value, err := Eval(stmt.Value, env)
	if err != nil {
		return nil, err
	}
	if value == nil {
		value = object.Null
	}
	if e, ok := value.(*object.Error); ok && len(e.Stack) == 0 {
		e.Stack = env.Runtime().Stack()
	}
	return nil, &panicSignal{value: value}
}

func runDefers(rt *Runtime, frame *callFrame) error {
	var regularErr error
	for i := len(frame.defers) - 1; i >= 0; i-- {
		err := invokeDeferredCall(rt, frame, frame.defers[i])
		if panicErr, ok := err.(*panicSignal); ok {
			frame.panic = panicErr
			regularErr = nil
			continue
		}
		if err != nil && frame.panic == nil && regularErr == nil {
			regularErr = err
		}
	}
	if frame.panic != nil {
		return frame.panic
	}
	return regularErr
}

func invokeDeferredCall(rt *Runtime, parent *callFrame, call deferredCall) error {
	if call.builtin != nil {
		_, err := call.builtin(call.pos, call.args)
		return err
	}
	if fn, ok := call.callable.(*Function); ok {
		frame := rt.pushFrame()
		frame.panic = parent.panic
		defer rt.popFrame()
		values, err := bindArguments(fn, call.args, call.pos)
		if err != nil {
			return err
		}
		_, err = applyFunctionBodyWithFrame(rt, fn, values, call.pos, frame)
		parent.panic = frame.panic
		return err
	}
	_, err := applyCallableWithPanic(rt, call.callable, call.args, call.pos, parent.panic)
	return err
}

func applyCallableWithPanic(rt *Runtime, callable object.Object, args []object.Object, pos ast.Position, inherited *panicSignal) (object.Object, error) {
	switch fn := callable.(type) {
	case *Class:
		return fn.instantiate(rt, pos, args)
	case tagsys.Callable:
		return callTagsysCallable(fn, args, pos)
	default:
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("%s 不是可调用对象", callable.Kind())}
	}
}

func evalRecover(rt *Runtime) object.Object {
	frame := rt.currentFrame()
	if frame == nil || frame.panic == nil {
		return object.Null
	}
	value := frame.panic.value
	frame.panic = nil
	return value
}
