package evaluator

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fan/internal/object"
)

func networkOptionString(options object.Object, key string) (string, bool, error) {
	return optionString(options, key)
}

func networkOptionInt(options object.Object, key string) (int64, bool, error) {
	if options == nil || options == object.Null {
		return 0, false, nil
	}
	dict, ok := options.(*object.Dict)
	if !ok {
		return 0, false, fmt.Errorf("请求选项必须是字典")
	}
	value, exists := dict.Get(&object.String{Value: key})
	if !exists || value == object.Null {
		return 0, false, nil
	}
	number, ok := value.(*object.Integer)
	if !ok {
		return 0, false, fmt.Errorf("选项 %s 必须是整数", key)
	}
	return number.Value, true, nil
}

func networkHeaders(options object.Object) (map[string]string, error) {
	headers := map[string]string{}
	if options == nil || options == object.Null {
		return headers, nil
	}
	dict, ok := options.(*object.Dict)
	if !ok {
		return nil, fmt.Errorf("请求选项必须是字典")
	}
	value, exists := dict.Get(&object.String{Value: "headers"})
	if !exists || value == object.Null {
		return headers, nil
	}
	headerDict, ok := value.(*object.Dict)
	if !ok {
		return nil, fmt.Errorf("选项 headers 必须是字典")
	}
	for _, key := range headerDict.Keys {
		name, ok := key.(*object.String)
		if !ok {
			return nil, fmt.Errorf("请求头名称必须是字符串")
		}
		headerValue, ok := headerDict.Get(key)
		if !ok {
			return nil, fmt.Errorf("请求头 %s 缺少值", name.Value)
		}
		text, ok := headerValue.(*object.String)
		if !ok {
			return nil, fmt.Errorf("请求头 %s 的值必须是字符串", name.Value)
		}
		headers[name.Value] = text.Value
	}
	return headers, nil
}

func makeHTTPResponse(resp *http.Response, body string) *object.Dict {
	result := object.NewDict()
	result.Set(&object.String{Value: "statusCode"}, &object.Integer{Value: int64(resp.StatusCode)})
	result.Set(&object.String{Value: "statusText"}, &object.String{Value: resp.Status})
	result.Set(&object.String{Value: "body"}, &object.String{Value: body})
	result.Set(&object.String{Value: "success"}, object.BoolOf(resp.StatusCode >= 200 && resp.StatusCode < 300))
	headers := object.NewDict()
	for name, values := range resp.Header {
		if len(values) > 0 {
			headers.Set(&object.String{Value: strings.ToLower(name)}, &object.String{Value: values[0]})
		}
	}
	result.Set(&object.String{Value: "headers"}, headers)
	return result
}

func failedHTTPResponse() *object.Dict {
	result := object.NewDict()
	result.Set(&object.String{Value: "statusCode"}, &object.Integer{Value: 0})
	result.Set(&object.String{Value: "statusText"}, &object.String{Value: ""})
	result.Set(&object.String{Value: "body"}, &object.String{Value: ""})
	result.Set(&object.String{Value: "success"}, object.False)
	result.Set(&object.String{Value: "headers"}, object.NewDict())
	return result
}

func nativeHTTPRequest(args []object.Object) ([]object.Object, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("httpRequest 至少需要 2 个参数")
	}
	method, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("httpRequest 第 1 个参数必须是字符串")
	}
	requestMethod := strings.ToUpper(strings.TrimSpace(method.Value))
	if requestMethod == "" || strings.ContainsAny(requestMethod, " \t\r\n") {
		return []object.Object{failedHTTPResponse(), object.NewError("HTTP 方法不合法")}, nil
	}
	address, ok := args[1].(*object.String)
	if !ok {
		return nil, fmt.Errorf("httpRequest 第 2 个参数必须是字符串")
	}
	parsedURL, err := url.Parse(address.Value)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return []object.Object{failedHTTPResponse(), object.NewError("URL 不合法")}, nil
	}
	var options object.Object = object.Null
	if len(args) >= 3 {
		options = args[2]
	}
	body, _, err := networkOptionString(options, "body")
	if err != nil {
		return nil, err
	}
	headers, err := networkHeaders(options)
	if err != nil {
		return nil, err
	}
	timeoutSeconds, timeoutExists, err := networkOptionInt(options, "timeout")
	if err != nil {
		return nil, err
	}
	timeout := 30 * time.Second
	if timeoutExists {
		if timeoutSeconds < 0 {
			return nil, fmt.Errorf("选项 timeout 不能小于 0")
		}
		timeout = time.Duration(timeoutSeconds) * time.Second
	}
	client := &http.Client{Timeout: timeout}
	request, err := http.NewRequest(requestMethod, address.Value, strings.NewReader(body))
	if err != nil {
		return []object.Object{failedHTTPResponse(), object.NewError("无法创建 HTTP 请求：" + err.Error())}, nil
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	resp, err := client.Do(request)
	if err != nil {
		return []object.Object{failedHTTPResponse(), object.NewError("HTTP 请求失败：" + err.Error())}, nil
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return []object.Object{failedHTTPResponse(), object.NewError("读取 HTTP 响应失败：" + err.Error())}, nil
	}
	return []object.Object{makeHTTPResponse(resp, string(data)), object.Null}, nil
}

func init() {
	RegisterNativeFunction("httpRequest", nativeHTTPRequest)
}
