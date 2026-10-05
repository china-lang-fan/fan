package evaluator

import (
	"fmt"
	"strconv"
	"strings"

	"fan/internal/object"
)

func nativeParseInt(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("parseInt 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("parseInt 参数必须是字符串")
	}
	number, err := strconv.ParseInt(strings.TrimSpace(value.Value), 10, 64)
	if err != nil {
		return []object.Object{&object.Integer{Value: 0}, object.NewError("无法解析为整数：" + value.Value)}, nil
	}
	return []object.Object{&object.Integer{Value: number}, object.Null}, nil
}

func nativeParseFloat(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("parseFloat 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("parseFloat 参数必须是字符串")
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(value.Value), 64)
	if err != nil {
		return []object.Object{&object.Float{Value: 0}, object.NewError("无法解析为小数：" + value.Value)}, nil
	}
	return []object.Object{&object.Float{Value: number}, object.Null}, nil
}

func nativeFloatToString(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("floatToString 需要 1 个参数")
	}
	value, ok := toFloatObject(args[0])
	if !ok {
		return nil, fmt.Errorf("floatToString 参数必须是数字")
	}
	return []object.Object{&object.String{Value: strconv.FormatFloat(value, 'g', -1, 64)}}, nil
}

func nativeParseBool(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("parseBool 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("parseBool 参数必须是字符串")
	}
	switch strings.ToLower(strings.TrimSpace(value.Value)) {
	case "真", "true", "1", "yes", "on":
		return []object.Object{object.True, object.Null}, nil
	case "假", "false", "0", "no", "off":
		return []object.Object{object.False, object.Null}, nil
	}
	return []object.Object{object.False, object.NewError("无法解析为布尔：" + value.Value)}, nil
}

func init() {
	RegisterNativeFunction("parseInt", nativeParseInt)
	RegisterNativeFunction("parseFloat", nativeParseFloat)
	RegisterNativeFunction("floatToString", nativeFloatToString)
	RegisterNativeFunction("parseBool", nativeParseBool)
}
