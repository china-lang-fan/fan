package evaluator

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"

	"fan/internal/object"
)

func objectToJSONValue(value object.Object) (interface{}, error) {
	switch v := value.(type) {
	case *object.Integer:
		return v.Value, nil
	case *object.Float:
		return v.Value, nil
	case *object.String:
		return v.Value, nil
	case *object.Bool:
		return v.Value, nil
	case *object.Nil:
		return nil, nil
	case *object.Array:
		out := make([]interface{}, 0, len(v.Elements))
		for _, element := range v.Elements {
			converted, err := objectToJSONValue(element)
			if err != nil {
				return nil, err
			}
			out = append(out, converted)
		}
		return out, nil
	case *object.Dict:
		out := map[string]interface{}{}
		for _, key := range v.Keys {
			text, ok := key.(*object.String)
			if !ok {
				return nil, fmt.Errorf("JSON 对象的键必须是字符串")
			}
			element, _ := v.Get(key)
			converted, err := objectToJSONValue(element)
			if err != nil {
				return nil, err
			}
			out[text.Value] = converted
		}
		return out, nil
	}
	return nil, fmt.Errorf("无法转换为 JSON：%s", value.Kind())
}

func jsonValueToObject(value interface{}) object.Object {
	switch v := value.(type) {
	case nil:
		return object.Null
	case float64:
		if float64(int64(v)) == v {
			return &object.Integer{Value: int64(v)}
		}
		return &object.Float{Value: v}
	case string:
		return &object.String{Value: v}
	case bool:
		return object.BoolOf(v)
	case []interface{}:
		elements := make([]object.Object, 0, len(v))
		for _, item := range v {
			elements = append(elements, jsonValueToObject(item))
		}
		return &object.Array{Elements: elements}
	case map[string]interface{}:
		dict := object.NewDict()
		for key, item := range v {
			dict.Set(&object.String{Value: key}, jsonValueToObject(item))
		}
		return dict
	}
	return object.Null
}

func nativeJSONEncode(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("jsonEncode 需要 1 个参数")
	}
	value, err := objectToJSONValue(args[0])
	if err != nil {
		return []object.Object{&object.String{Value: ""}, object.NewError(err.Error())}, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return []object.Object{&object.String{Value: ""}, object.NewError(err.Error())}, nil
	}
	return []object.Object{&object.String{Value: string(data)}, object.Null}, nil
}

func nativeJSONDecode(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("jsonDecode 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("jsonDecode 参数必须是字符串")
	}
	var result interface{}
	if err := json.Unmarshal([]byte(value.Value), &result); err != nil {
		return []object.Object{object.Null, object.NewError("无法解析 JSON：" + err.Error())}, nil
	}
	return []object.Object{jsonValueToObject(result), object.Null}, nil
}

func nativeBase64Encode(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("base64Encode 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("base64Encode 参数必须是字符串")
	}
	return []object.Object{&object.String{Value: base64.StdEncoding.EncodeToString([]byte(value.Value))}}, nil
}

func nativeBase64Decode(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("base64Decode 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("base64Decode 参数必须是字符串")
	}
	data, err := base64.StdEncoding.DecodeString(value.Value)
	if err != nil {
		return []object.Object{&object.String{Value: ""}, object.NewError("无法解码 Base64：" + err.Error())}, nil
	}
	return []object.Object{&object.String{Value: string(data)}, object.Null}, nil
}

func nativeURLEncode(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("urlEncode 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("urlEncode 参数必须是字符串")
	}
	return []object.Object{&object.String{Value: url.QueryEscape(value.Value)}}, nil
}

func nativeURLDecode(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("urlDecode 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("urlDecode 参数必须是字符串")
	}
	decoded, err := url.QueryUnescape(value.Value)
	if err != nil {
		return []object.Object{&object.String{Value: ""}, object.NewError("无法解码 URL：" + err.Error())}, nil
	}
	return []object.Object{&object.String{Value: decoded}, object.Null}, nil
}

func nativeHexEncode(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("hexEncode 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("hexEncode 参数必须是字符串")
	}
	return []object.Object{&object.String{Value: hex.EncodeToString([]byte(value.Value))}}, nil
}

func nativeHexDecode(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("hexDecode 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("hexDecode 参数必须是字符串")
	}
	data, err := hex.DecodeString(value.Value)
	if err != nil {
		return []object.Object{&object.String{Value: ""}, object.NewError("无法解码十六进制：" + err.Error())}, nil
	}
	return []object.Object{&object.String{Value: string(data)}, object.Null}, nil
}

func init() {
	RegisterNativeFunction("jsonEncode", nativeJSONEncode)
	RegisterNativeFunction("jsonDecode", nativeJSONDecode)
	RegisterNativeFunction("base64Encode", nativeBase64Encode)
	RegisterNativeFunction("base64Decode", nativeBase64Decode)
	RegisterNativeFunction("urlEncode", nativeURLEncode)
	RegisterNativeFunction("urlDecode", nativeURLDecode)
	RegisterNativeFunction("hexEncode", nativeHexEncode)
	RegisterNativeFunction("hexDecode", nativeHexDecode)
}
