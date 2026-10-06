package evaluator

import (
	"fmt"
	"regexp"

	"fan/internal/object"
)

func compileRegexPattern(args []object.Object, required int, name string) (*regexp.Regexp, error) {
	if len(args) < required {
		return nil, fmt.Errorf("%s 需要 %d 个参数", name, required)
	}
	pattern, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("%s 第 1 个参数必须是字符串", name)
	}
	compiled, err := regexp.Compile(pattern.Value)
	if err != nil {
		return nil, fmt.Errorf("正则表达式不合法：%s", err.Error())
	}
	return compiled, nil
}

func regexTextArg(args []object.Object, index int, name string) (string, error) {
	if len(args) <= index {
		return "", fmt.Errorf("%s 第 %d 个参数必须是字符串", name, index+1)
	}
	value, ok := args[index].(*object.String)
	if !ok {
		return "", fmt.Errorf("%s 第 %d 个参数必须是字符串", name, index+1)
	}
	return value.Value, nil
}

func regexIntArg(args []object.Object, index int, name string) (int, error) {
	if len(args) <= index {
		return -1, nil
	}
	value, ok := args[index].(*object.Integer)
	if !ok {
		return 0, fmt.Errorf("%s 第 %d 个参数必须是整数", name, index+1)
	}
	return int(value.Value), nil
}

func stringArray(values []string) *object.Array {
	elements := make([]object.Object, 0, len(values))
	for _, value := range values {
		elements = append(elements, &object.String{Value: value})
	}
	return &object.Array{Elements: elements}
}

func nativeRegexMatch(args []object.Object) ([]object.Object, error) {
	pattern, err := compileRegexPattern(args, 2, "regexMatch")
	if err != nil {
		return []object.Object{object.False, object.NewError(err.Error())}, nil
	}
	text, err := regexTextArg(args, 1, "regexMatch")
	if err != nil {
		return nil, err
	}
	return []object.Object{object.BoolOf(pattern.MatchString(text)), object.Null}, nil
}

func nativeRegexFind(args []object.Object) ([]object.Object, error) {
	pattern, err := compileRegexPattern(args, 2, "regexFind")
	if err != nil {
		return []object.Object{&object.String{Value: ""}, object.NewError(err.Error())}, nil
	}
	text, err := regexTextArg(args, 1, "regexFind")
	if err != nil {
		return nil, err
	}
	return []object.Object{&object.String{Value: pattern.FindString(text)}, object.Null}, nil
}

func nativeRegexFindAll(args []object.Object) ([]object.Object, error) {
	pattern, err := compileRegexPattern(args, 2, "regexFindAll")
	if err != nil {
		return []object.Object{&object.Array{}, object.NewError(err.Error())}, nil
	}
	text, err := regexTextArg(args, 1, "regexFindAll")
	if err != nil {
		return nil, err
	}
	limit, err := regexIntArg(args, 2, "regexFindAll")
	if err != nil {
		return nil, err
	}
	return []object.Object{stringArray(pattern.FindAllString(text, limit)), object.Null}, nil
}

func matchGroups(pattern *regexp.Regexp, text string) []object.Object {
	match := pattern.FindStringSubmatchIndex(text)
	if match == nil {
		return []object.Object{}
	}
	groups := make([]object.Object, 0, len(match)/2)
	for i := 0; i < len(match); i += 2 {
		if match[i] < 0 {
			groups = append(groups, object.Null)
		} else {
			groups = append(groups, &object.String{Value: text[match[i]:match[i+1]]})
		}
	}
	return groups
}

func nativeRegexFindGroups(args []object.Object) ([]object.Object, error) {
	pattern, err := compileRegexPattern(args, 2, "regexFindGroups")
	if err != nil {
		return []object.Object{&object.Array{}, object.NewError(err.Error())}, nil
	}
	text, err := regexTextArg(args, 1, "regexFindGroups")
	if err != nil {
		return nil, err
	}
	return []object.Object{&object.Array{Elements: matchGroups(pattern, text)}, object.Null}, nil
}

func nativeRegexFindAllGroups(args []object.Object) ([]object.Object, error) {
	pattern, err := compileRegexPattern(args, 2, "regexFindAllGroups")
	if err != nil {
		return []object.Object{&object.Array{}, object.NewError(err.Error())}, nil
	}
	text, err := regexTextArg(args, 1, "regexFindAllGroups")
	if err != nil {
		return nil, err
	}
	limit, err := regexIntArg(args, 2, "regexFindAllGroups")
	if err != nil {
		return nil, err
	}
	matches := pattern.FindAllStringSubmatchIndex(text, limit)
	results := make([]object.Object, 0, len(matches))
	for _, match := range matches {
		groups := make([]object.Object, 0, len(match)/2)
		for i := 0; i < len(match); i += 2 {
			if match[i] < 0 {
				groups = append(groups, object.Null)
			} else {
				groups = append(groups, &object.String{Value: text[match[i]:match[i+1]]})
			}
		}
		results = append(results, &object.Array{Elements: groups})
	}
	return []object.Object{&object.Array{Elements: results}, object.Null}, nil
}

func nativeRegexReplace(args []object.Object) ([]object.Object, error) {
	pattern, err := compileRegexPattern(args, 3, "regexReplace")
	if err != nil {
		return []object.Object{&object.String{Value: ""}, object.NewError(err.Error())}, nil
	}
	text, err := regexTextArg(args, 1, "regexReplace")
	if err != nil {
		return nil, err
	}
	replacement, err := regexTextArg(args, 2, "regexReplace")
	if err != nil {
		return nil, err
	}
	limit, err := regexIntArg(args, 3, "regexReplace")
	if err != nil {
		return nil, err
	}
	matches := pattern.FindAllStringSubmatchIndex(text, limit)
	output := make([]byte, 0, len(text))
	last := 0
	for _, match := range matches {
		output = append(output, text[last:match[0]]...)
		output = pattern.ExpandString(output, replacement, text, match)
		last = match[1]
	}
	output = append(output, text[last:]...)
	return []object.Object{&object.String{Value: string(output)}, object.Null}, nil
}

func nativeRegexSplit(args []object.Object) ([]object.Object, error) {
	pattern, err := compileRegexPattern(args, 2, "regexSplit")
	if err != nil {
		return []object.Object{&object.Array{}, object.NewError(err.Error())}, nil
	}
	text, err := regexTextArg(args, 1, "regexSplit")
	if err != nil {
		return nil, err
	}
	limit, err := regexIntArg(args, 2, "regexSplit")
	if err != nil {
		return nil, err
	}
	return []object.Object{stringArray(pattern.Split(text, limit)), object.Null}, nil
}

func nativeRegexQuote(args []object.Object) ([]object.Object, error) {
	text, err := regexTextArg(args, 0, "regexQuote")
	if err != nil {
		return nil, err
	}
	return []object.Object{&object.String{Value: regexp.QuoteMeta(text)}}, nil
}

func init() {
	RegisterNativeFunction("regexMatch", nativeRegexMatch)
	RegisterNativeFunction("regexFind", nativeRegexFind)
	RegisterNativeFunction("regexFindAll", nativeRegexFindAll)
	RegisterNativeFunction("regexFindGroups", nativeRegexFindGroups)
	RegisterNativeFunction("regexFindAllGroups", nativeRegexFindAllGroups)
	RegisterNativeFunction("regexReplace", nativeRegexReplace)
	RegisterNativeFunction("regexSplit", nativeRegexSplit)
	RegisterNativeFunction("regexQuote", nativeRegexQuote)
}
