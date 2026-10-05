package evaluator

import (
	"fmt"
	"math/rand"

	"fan/internal/object"
)

func nativeRandomInt(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("randomInt 需要 1 个参数")
	}
	value, ok := args[0].(*object.Integer)
	if !ok || value.Value <= 0 {
		return nil, fmt.Errorf("randomInt 参数必须是正整数")
	}
	return []object.Object{&object.Integer{Value: rand.Int63n(value.Value)}}, nil
}

func nativeRandomFloat(_ []object.Object) ([]object.Object, error) {
	return []object.Object{&object.Float{Value: rand.Float64()}}, nil
}

func init() {
	RegisterNativeFunction("randomInt", nativeRandomInt)
	RegisterNativeFunction("randomFloat", nativeRandomFloat)
}
