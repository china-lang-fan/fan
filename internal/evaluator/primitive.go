package evaluator

import (
	"fmt"
	"math"
	"unicode/utf8"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/tagsys"
)

func nativeFromBuiltin(fn builtinFn) NativeFunction {
	return func(args []object.Object) ([]object.Object, error) {
		value, err := fn(ast.Position{}, args)
		if err != nil {
			return nil, err
		}
		return []object.Object{value}, nil
	}
}

func init() {
	RegisterNativeFunction("type", nativeFromBuiltin(builtinType))
	RegisterNativeFunction("trunc", nativeFromBuiltin(builtinTrunc))
	RegisterNativeFunction("ord", nativeFromBuiltin(builtinOrd))
	RegisterNativeFunction("char", nativeFromBuiltin(builtinChar))
	RegisterNativeFunction("error", nativeFromBuiltin(builtinError))
	RegisterNativeFunction("len", nativeFromBuiltin(builtinLength))
	RegisterNativeFunction("append", nativeFromBuiltin(builtinAppend))
	RegisterNativeFunction("fail", nativeFromBuiltin(builtinFail))
}

func targetIsVariadic(target tagsys.Target) bool {
	fn, ok := target.(*Function)
	if !ok {
		return false
	}
	for _, param := range fn.Params {
		if param.Variadic {
			return true
		}
	}
	return false
}

func builtinType(pos ast.Position, args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("type 需要 1 个参数，实际 %d 个", len(args))}
	}
	return &object.String{Value: runtimeTypeName(args[0])}, nil
}

func runtimeTypeName(value object.Object) string {
	switch value.(type) {
	case *object.Integer:
		return "integer"
	case *object.Float:
		return "float"
	case *object.String:
		return "string"
	case *object.Bool:
		return "boolean"
	case *object.Nil:
		return "nil"
	case *object.Array:
		return "array"
	case *object.Dict:
		return "dict"
	case *object.Error:
		return "error"
	case *object.Tuple:
		return "tuple"
	case *object.Module:
		return "module"
	case *Instance:
		return "instance"
	case *Function:
		return "function"
	case *Class:
		return "class"
	}
	return "unknown"
}

func builtinTrunc(pos ast.Position, args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("trunc 需要 1 个参数，实际 %d 个", len(args))}
	}
	switch value := args[0].(type) {
	case *object.Integer:
		return value, nil
	case *object.Float:
		return &object.Integer{Value: int64(math.Trunc(value.Value))}, nil
	}
	return nil, &EvalError{Pos: pos, Reason: "trunc 参数必须是数字"}
}

func builtinOrd(pos ast.Position, args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("ord 需要 1 个参数，实际 %d 个", len(args))}
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, &EvalError{Pos: pos, Reason: "ord 参数必须是字符串"}
	}
	r, size := utf8.DecodeRuneInString(value.Value)
	if size == 0 || r == utf8.RuneError {
		return nil, &EvalError{Pos: pos, Reason: "ord 参数必须包含一个字符"}
	}
	if len(value.Value) != size {
		return nil, &EvalError{Pos: pos, Reason: "ord 参数必须包含一个字符"}
	}
	return &object.Integer{Value: int64(r)}, nil
}

func builtinChar(pos ast.Position, args []object.Object) (object.Object, error) {
	if len(args) != 1 {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("char 需要 1 个参数，实际 %d 个", len(args))}
	}
	value, ok := args[0].(*object.Integer)
	if !ok {
		return nil, &EvalError{Pos: pos, Reason: "char 参数必须是整数"}
	}
	r := rune(value.Value)
	if !utf8.ValidRune(r) {
		return nil, &EvalError{Pos: pos, Reason: "char 参数不是有效的 Unicode 编码"}
	}
	return &object.String{Value: string(r)}, nil
}
