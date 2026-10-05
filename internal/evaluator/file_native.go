package evaluator

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"fan/internal/object"
)

func errObjectOrNull(err error) object.Object {
	if err == nil {
		return object.Null
	}
	return object.NewError(err.Error())
}

func requireStringArg(name string, args []object.Object, index int) (string, bool) {
	if len(args) <= index {
		return "", false
	}
	value, ok := args[index].(*object.String)
	if !ok {
		return "", false
	}
	return value.Value, true
}

func nativeFileRead(args []object.Object) ([]object.Object, error) {
	path, ok := requireStringArg("readFile", args, 0)
	if !ok {
		return nil, fmt.Errorf("readFile 第 1 个参数必须是字符串路径")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return []object.Object{&object.String{Value: ""}, object.NewError(err.Error())}, nil
	}
	return []object.Object{&object.String{Value: string(data)}, object.Null}, nil
}

func nativeFileWrite(args []object.Object) ([]object.Object, error) {
	path, ok := requireStringArg("writeFile", args, 0)
	if !ok {
		return nil, fmt.Errorf("writeFile 第 1 个参数必须是字符串路径")
	}
	content, ok := requireStringArg("writeFile", args, 1)
	if !ok {
		return nil, fmt.Errorf("writeFile 第 2 个参数必须是字符串")
	}
	err := os.WriteFile(path, []byte(content), 0644)
	return []object.Object{errObjectOrNull(err)}, nil
}

func nativeFileAppend(args []object.Object) ([]object.Object, error) {
	path, ok := requireStringArg("appendFile", args, 0)
	if !ok {
		return nil, fmt.Errorf("appendFile 第 1 个参数必须是字符串路径")
	}
	content, ok := requireStringArg("appendFile", args, 1)
	if !ok {
		return nil, fmt.Errorf("appendFile 第 2 个参数必须是字符串")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer file.Close()
		_, err = file.WriteString(content)
	}
	return []object.Object{errObjectOrNull(err)}, nil
}

func nativeFileRemove(args []object.Object) ([]object.Object, error) {
	path, ok := requireStringArg("remove", args, 0)
	if !ok {
		return nil, fmt.Errorf("remove 第 1 个参数必须是字符串路径")
	}
	err := os.Remove(path)
	return []object.Object{errObjectOrNull(err)}, nil
}

func nativeFileRemoveAll(args []object.Object) ([]object.Object, error) {
	path, ok := requireStringArg("removeAll", args, 0)
	if !ok {
		return nil, fmt.Errorf("removeAll 第 1 个参数必须是字符串路径")
	}
	err := os.RemoveAll(path)
	return []object.Object{errObjectOrNull(err)}, nil
}

func nativeFileRename(args []object.Object) ([]object.Object, error) {
	oldPath, ok := requireStringArg("rename", args, 0)
	if !ok {
		return nil, fmt.Errorf("rename 第 1 个参数必须是字符串路径")
	}
	newPath, ok := requireStringArg("rename", args, 1)
	if !ok {
		return nil, fmt.Errorf("rename 第 2 个参数必须是字符串路径")
	}
	err := os.Rename(oldPath, newPath)
	return []object.Object{errObjectOrNull(err)}, nil
}

func nativeFileMakeDir(args []object.Object) ([]object.Object, error) {
	path, ok := requireStringArg("makeDir", args, 0)
	if !ok {
		return nil, fmt.Errorf("makeDir 第 1 个参数必须是字符串路径")
	}
	recursive := false
	if len(args) > 1 {
		if flag, ok := args[1].(*object.Bool); ok {
			recursive = flag.Value
		}
	}
	var err error
	if recursive {
		err = os.MkdirAll(path, 0755)
	} else {
		err = os.Mkdir(path, 0755)
	}
	return []object.Object{errObjectOrNull(err)}, nil
}

func nativeFileList(args []object.Object) ([]object.Object, error) {
	path, ok := requireStringArg("listDir", args, 0)
	if !ok {
		return nil, fmt.Errorf("listDir 第 1 个参数必须是字符串路径")
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return []object.Object{&object.Array{}, object.NewError(err.Error())}, nil
	}
	elements := make([]object.Object, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += string(filepath.Separator)
		}
		elements = append(elements, &object.String{Value: name})
	}
	return []object.Object{&object.Array{Elements: elements}, object.Null}, nil
}

func nativeFileStat(args []object.Object) ([]object.Object, error) {
	path, ok := requireStringArg("stat", args, 0)
	if !ok {
		return nil, fmt.Errorf("stat 第 1 个参数必须是字符串路径")
	}
	info, err := os.Stat(path)
	if err != nil {
		return []object.Object{object.Null, object.NewError(err.Error())}, nil
	}
	dict := object.NewDict()
	put := func(key string, value object.Object) {
		dict.Set(&object.String{Value: key}, value)
	}
	put("name", &object.String{Value: info.Name()})
	put("size", &object.Integer{Value: info.Size()})
	put("isDir", object.BoolOf(info.IsDir()))
	put("mode", &object.String{Value: info.Mode().String()})
	put("modified", &object.String{Value: info.ModTime().Format(time.RFC3339)})
	return []object.Object{dict, object.Null}, nil
}

func nativeFileExists(args []object.Object) ([]object.Object, error) {
	path, ok := requireStringArg("exists", args, 0)
	if !ok {
		return nil, fmt.Errorf("exists 第 1 个参数必须是字符串路径")
	}
	_, err := os.Stat(path)
	return []object.Object{object.BoolOf(err == nil)}, nil
}

func nativeTempDir(args []object.Object) ([]object.Object, error) {
	prefix := ""
	if len(args) > 0 {
		if value, ok := args[0].(*object.String); ok {
			prefix = value.Value
		}
	}
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return []object.Object{&object.String{Value: ""}, object.NewError(err.Error())}, nil
	}
	return []object.Object{&object.String{Value: dir}, object.Null}, nil
}

func nativeTempFile(args []object.Object) ([]object.Object, error) {
	prefix := ""
	if len(args) > 0 {
		if value, ok := args[0].(*object.String); ok {
			prefix = value.Value
		}
	}
	file, err := os.CreateTemp("", prefix)
	if err != nil {
		return []object.Object{&object.String{Value: ""}, object.NewError(err.Error())}, nil
	}
	path := file.Name()
	file.Close()
	return []object.Object{&object.String{Value: path}, object.Null}, nil
}

func nativePathSeparator(_ []object.Object) ([]object.Object, error) {
	return []object.Object{&object.String{Value: string(filepath.Separator)}}, nil
}

func nativePathBase(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("pathBase 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("pathBase 参数必须是字符串")
	}
	return []object.Object{&object.String{Value: filepath.Base(value.Value)}}, nil
}

func nativePathExt(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("pathExt 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("pathExt 参数必须是字符串")
	}
	return []object.Object{&object.String{Value: filepath.Ext(value.Value)}}, nil
}

func nativePathDir(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("pathDir 需要 1 个参数")
	}
	value, ok := args[0].(*object.String)
	if !ok {
		return nil, fmt.Errorf("pathDir 参数必须是字符串")
	}
	return []object.Object{&object.String{Value: filepath.Dir(value.Value)}}, nil
}

func nativePathJoin(args []object.Object) ([]object.Object, error) {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		if elements, ok := arg.(*object.Array); ok {
			for _, element := range elements.Elements {
				value, ok := element.(*object.String)
				if !ok {
					return nil, fmt.Errorf("pathJoin 参数必须是字符串")
				}
				parts = append(parts, value.Value)
			}
			continue
		}
		value, ok := arg.(*object.String)
		if !ok {
			return nil, fmt.Errorf("pathJoin 参数必须是字符串")
		}
		parts = append(parts, value.Value)
	}
	return []object.Object{&object.String{Value: filepath.Join(parts...)}}, nil
}

func nativeIntToString(args []object.Object) ([]object.Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("intToString 需要 1 个参数")
	}
	value, ok := args[0].(*object.Integer)
	if !ok {
		return nil, fmt.Errorf("intToString 参数必须是整数")
	}
	return []object.Object{&object.String{Value: strconv.FormatInt(value.Value, 10)}}, nil
}

func init() {
	RegisterNativeFunction("readFile", nativeFileRead)
	RegisterNativeFunction("writeFile", nativeFileWrite)
	RegisterNativeFunction("appendFile", nativeFileAppend)
	RegisterNativeFunction("remove", nativeFileRemove)
	RegisterNativeFunction("removeAll", nativeFileRemoveAll)
	RegisterNativeFunction("rename", nativeFileRename)
	RegisterNativeFunction("makeDir", nativeFileMakeDir)
	RegisterNativeFunction("listDir", nativeFileList)
	RegisterNativeFunction("stat", nativeFileStat)
	RegisterNativeFunction("exists", nativeFileExists)
	RegisterNativeFunction("tempDir", nativeTempDir)
	RegisterNativeFunction("tempFile", nativeTempFile)
	RegisterNativeFunction("pathSeparator", nativePathSeparator)
	RegisterNativeFunction("pathJoin", nativePathJoin)
	RegisterNativeFunction("pathBase", nativePathBase)
	RegisterNativeFunction("pathExt", nativePathExt)
	RegisterNativeFunction("pathDir", nativePathDir)
	RegisterNativeFunction("intToString", nativeIntToString)
}
