package evaluator

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"fan/internal/object"
)

func nativeParseInt(_ *Runtime, args []object.Object) ([]object.Object, error) {
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

func nativeParseFloat(_ *Runtime, args []object.Object) ([]object.Object, error) {
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

func nativeFloatToString(_ *Runtime, args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("floatToString 需要 1 个参数")
	}
	value, ok := toFloatObject(args[0])
	if !ok {
		return nil, fmt.Errorf("floatToString 参数必须是数字")
	}
	return []object.Object{&object.String{Value: strconv.FormatFloat(value, 'g', -1, 64)}}, nil
}

func nativeParseBool(_ *Runtime, args []object.Object) ([]object.Object, error) {
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

func nativeToByte(_ *Runtime, args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("toByte 需要 1 个参数")
	}
	v, ok := intValue(args[0])
	if !ok {
		return nil, fmt.Errorf("toByte 参数必须是整数或字节，实际是 %s", args[0].Kind())
	}
	if v < 0 || v > 255 {
		return []object.Object{object.Null, object.NewErrorf("字节超出范围：%d（允许 0–255）", v)}, nil
	}
	return []object.Object{&object.Byte{Value: byte(v)}, object.Null}, nil
}

func nativeToChar(_ *Runtime, args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("toChar 需要 1 个参数")
	}
	switch v := args[0].(type) {
	case *object.Char:
		return []object.Object{v, object.Null}, nil
	case *object.String:
		runes := []rune(v.Value)
		if len(runes) != 1 {
			return []object.Object{object.Null, object.NewErrorf("字符串必须包含恰好一个字符，实际 %d 个", len(runes))}, nil
		}
		return []object.Object{&object.Char{Value: runes[0]}, object.Null}, nil
	case *object.Integer:
		r := rune(v.Value)
		if !utf8.ValidRune(r) {
			return []object.Object{object.Null, object.NewErrorf("整数 %d 不是有效的 Unicode 码点", v.Value)}, nil
		}
		return []object.Object{&object.Char{Value: r}, object.Null}, nil
	case *object.Byte:
		return []object.Object{&object.Char{Value: rune(v.Value)}, object.Null}, nil
	}
	return nil, fmt.Errorf("toChar 参数必须是整数、字节、字符或单字符串，实际是 %s", args[0].Kind())
}

func init() {
	RegisterNativeFunction("parseInt", nativeParseInt)
	RegisterNativeFunction("parseFloat", nativeParseFloat)
	RegisterNativeFunction("floatToString", nativeFloatToString)
	RegisterNativeFunction("parseBool", nativeParseBool)
	RegisterNativeFunction("toByte", nativeToByte)
	RegisterNativeFunction("toChar", nativeToChar)
}
