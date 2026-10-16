package evaluator

import (
	"fmt"
	"io"
	"strings"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/tagsys"
)

type builtinFn func(pos ast.Position, args []object.Object) (object.Object, error)

var builtins = map[string]builtinFn{
	"长度":     builtinLength,
	"追加":     builtinAppend,
	"打印":     nil,
	"错误":     builtinError,
	"注册标签处理": nil,
	"print":  nil,
	"error":  builtinError,
	"type":   builtinType,
	"trunc":  builtinTrunc,
	"ord":    builtinOrd,
	"char":   builtinChar,
	"fail":   builtinFail,
}

// runtimeBuiltin 处理需要访问运行实例的内建函数（打印/注册标签处理）。
func runtimeBuiltin(rt *Runtime, name string, pos ast.Position, args []object.Object) (object.Object, bool, error) {
	switch name {
	case "打印", "print":
		writeObjects(rt.Stdout(), args)
		return object.Null, true, nil
	case "注册标签处理":
		if len(args) != 2 {
			return nil, true, &EvalError{Pos: pos, Reason: "注册标签处理 需要 2 个参数"}
		}
		tagName, ok := args[0].(*object.String)
		if !ok {
			return nil, true, &EvalError{Pos: pos, Reason: "标签名必须是字符串"}
		}
		fn, ok := args[1].(*Function)
		if !ok {
			return nil, true, &EvalError{Pos: pos, Reason: "标签处理器必须是函数"}
		}
		rt.registerFanTagHandler(tagName.Value, fn)
		return object.Null, true, nil
	}
	return nil, false, nil
}

func IsBuiltin(name string) bool {
	_, ok := builtins[name]
	return ok
}

func builtinLength(pos ast.Position, args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("长度 需要 1 个参数，实际 %d 个", len(args))}
	}
	switch v := args[0].(type) {
	case *object.Array:
		return &object.Integer{Value: int64(len(v.Elements))}, nil
	case *object.Dict:
		return &object.Integer{Value: int64(v.Len())}, nil
	case *object.String:
		return &object.Integer{Value: int64(len([]rune(v.Value)))}, nil
	}
	return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("长度 不支持 %s", args[0].Kind())}
}

func builtinAppend(pos ast.Position, args []object.Object) (object.Object, error) {
	if len(args) < 2 {
		return nil, &EvalError{Pos: pos, Reason: "追加 至少需要 2 个参数"}
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("追加 的第 1 个参数必须是数组，实际是 %s", args[0].Kind())}
	}
	elems := make([]object.Object, 0, len(arr.Elements)+len(args)-1)
	elems = append(elems, arr.Elements...)
	elems = append(elems, args[1:]...)
	return &object.Array{Elements: elems}, nil
}

// writeObjects 将对象按空格连接并输出一行。
func writeObjects(w io.Writer, args []object.Object) {
	parts := make([]string, 0, len(args))
	for _, a := range args {
		parts = append(parts, object.Format(a))
	}
	fmt.Fprintln(w, strings.Join(parts, " "))
}

func builtinError(pos ast.Position, args []object.Object) (object.Object, error) {
	if len(args) == 0 {
		return object.NewError(""), nil
	}
	e := object.NewError(object.Format(args[0]))
	if len(args) >= 2 {
		if code, ok := args[1].(*object.String); ok {
			e.Code = code.Value
		} else if args[1] != object.Null {
			return nil, &EvalError{Pos: pos, Reason: "错误 的第 2 个参数（错误码）必须是字符串"}
		}
	}
	if len(args) >= 3 {
		switch details := args[2].(type) {
		case *object.Dict:
			e.Details = details
		case *object.Nil:
		default:
			return nil, &EvalError{Pos: pos, Reason: "错误 的第 3 个参数（详情）必须是字典"}
		}
	}
	return e, nil
}

func builtinFail(pos ast.Position, args []object.Object) (object.Object, error) {
	if len(args) == 0 {
		return nil, &EvalError{Pos: pos, Reason: "断言失败"}
	}
	return nil, &EvalError{Pos: pos, Reason: object.Format(args[0])}
}

func spreadTuples(args []object.Object) []object.Object {
	hasTuple := false
	for _, a := range args {
		if _, ok := a.(*object.Tuple); ok {
			hasTuple = true
			break
		}
	}
	if !hasTuple {
		return args
	}
	out := make([]object.Object, 0, len(args))
	for _, a := range args {
		if t, ok := a.(*object.Tuple); ok {
			out = append(out, t.Values...)
			continue
		}
		out = append(out, a)
	}
	return out
}

func evalCallExpr(node *ast.CallExpr, env *Environment) (object.Object, error) {
	ident, ok := node.Callee.(*ast.Identifier)
	if !ok {
		return nil, &EvalError{Pos: node.Position, Reason: "目前只支持调用内建函数或命名函数"}
	}
	args := make([]object.Object, 0, len(node.Args))
	for _, a := range node.Args {
		val, err := Eval(a, env)
		if err != nil {
			return nil, err
		}
		if val == nil {
			val = object.Null
		}
		args = append(args, val)
	}
	args = spreadTuples(args)
	if value, handled, err := runtimeBuiltin(env.Runtime(), ident.Name, node.Position, args); handled {
		return value, err
	}
	if fn, ok := builtins[ident.Name]; ok {
		return fn(node.Position, args)
	}
	val, ok := env.get(ident.Name)
	if !ok {
		return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("未知函数：%s", ident.Name)}
	}
	switch callable := val.(type) {
	case *Function:
		return applyFunction(callable, args, node.Position)
	case *Class:
		return callable.instantiate(env.Runtime(), node.Position, args)
	case *NamedType:
		if len(args) != 1 {
			return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("类型转换 %s 需要 1 个参数，实际 %d 个", callable.Name, len(args))}
		}
		return convertToNamed(callable, args[0], node.Position)
	case tagsys.Callable:
		return callTagsysCallable(callable, args, node.Position)
	}
	return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 不是函数或类", ident.Name)}
}
