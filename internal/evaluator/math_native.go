package evaluator

import (
	"fmt"
	"math"

	"fan/internal/object"
)

func toFloatObject(value object.Object) (float64, bool) {
	switch v := value.(type) {
	case *object.Integer:
		return float64(v.Value), true
	case *object.Float:
		return v.Value, true
	}
	return 0, false
}

func nativeSqrt(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sqrt 需要 1 个参数")
	}
	value, ok := toFloatObject(args[0])
	if !ok {
		return nil, fmt.Errorf("sqrt 参数必须是数字")
	}
	if value < 0 {
		return nil, fmt.Errorf("sqrt 参数不能为负数")
	}
	return []object.Object{&object.Float{Value: math.Sqrt(value)}}, nil
}

func nativePow(args []object.Object) ([]object.Object, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("pow 需要 2 个参数")
	}
	base, ok := toFloatObject(args[0])
	if !ok {
		return nil, fmt.Errorf("pow 第 1 个参数必须是数字")
	}
	exponent, ok := toFloatObject(args[1])
	if !ok {
		return nil, fmt.Errorf("pow 第 2 个参数必须是数字")
	}
	return []object.Object{&object.Float{Value: math.Pow(base, exponent)}}, nil
}

func init() {
	RegisterNativeFunction("sqrt", nativeSqrt)
	RegisterNativeFunction("pow", nativePow)
}
