package evaluator

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"fan/internal/object"
)

type processContext struct {
	scriptPath string
	args       []string
}

var (
	processStateMu        sync.RWMutex
	currentProcessContext = processContext{args: []string{}}
)

func SetProcessContext(scriptPath string, args []string) {
	copied := make([]string, len(args))
	copy(copied, args)
	processStateMu.Lock()
	currentProcessContext = processContext{scriptPath: scriptPath, args: copied}
	processStateMu.Unlock()
}

func ResetProcessContext() {
	processStateMu.Lock()
	currentProcessContext = processContext{args: []string{}}
	processStateMu.Unlock()
}

func getProcessContext() processContext {
	processStateMu.RLock()
	defer processStateMu.RUnlock()
	args := make([]string, len(currentProcessContext.args))
	copy(args, currentProcessContext.args)
	return processContext{scriptPath: currentProcessContext.scriptPath, args: args}
}

func requireProcessString(name string, args []object.Object, index int) (string, error) {
	if len(args) <= index {
		return "", fmt.Errorf("%s 需要 %d 个参数", name, index+1)
	}
	value, ok := args[index].(*object.String)
	if !ok {
		return "", fmt.Errorf("%s 第 %d 个参数必须是字符串", name, index+1)
	}
	return value.Value, nil
}

func optionString(options object.Object, key string) (string, bool, error) {
	if options == nil || options == object.Null {
		return "", false, nil
	}
	dict, ok := options.(*object.Dict)
	if !ok {
		return "", false, fmt.Errorf("执行选项必须是字典")
	}
	value, exists := dict.Get(&object.String{Value: key})
	if !exists || value == object.Null {
		return "", false, nil
	}
	text, ok := value.(*object.String)
	if !ok {
		return "", false, fmt.Errorf("选项 %s 必须是字符串", key)
	}
	return text.Value, true, nil
}

func nativeProcessArgs(_ []object.Object) ([]object.Object, error) {
	ctx := getProcessContext()
	elements := make([]object.Object, 0, len(ctx.args))
	for _, arg := range ctx.args {
		elements = append(elements, &object.String{Value: arg})
	}
	return []object.Object{&object.Array{Elements: elements}}, nil
}

func nativeProcessScript(_ []object.Object) ([]object.Object, error) {
	return []object.Object{&object.String{Value: getProcessContext().scriptPath}}, nil
}

func nativeProcessEnv(args []object.Object) ([]object.Object, error) {
	name, err := requireProcessString("env", args, 0)
	if err != nil {
		return nil, err
	}
	value := os.Getenv(name)
	return []object.Object{&object.String{Value: value}}, nil
}

func nativeProcessSetEnv(args []object.Object) ([]object.Object, error) {
	name, err := requireProcessString("setEnv", args, 0)
	if err != nil {
		return nil, err
	}
	value, err := requireProcessString("setEnv", args, 1)
	if err != nil {
		return nil, err
	}
	if err := os.Setenv(name, value); err != nil {
		return []object.Object{object.NewError(err.Error())}, nil
	}
	return []object.Object{object.Null}, nil
}

func nativeProcessWorkingDir(_ []object.Object) ([]object.Object, error) {
	dir, err := os.Getwd()
	if err != nil {
		return []object.Object{&object.String{Value: ""}, object.NewError(err.Error())}, nil
	}
	return []object.Object{&object.String{Value: dir}}, nil
}

func nativeProcessChangeDir(args []object.Object) ([]object.Object, error) {
	path, err := requireProcessString("changeDir", args, 0)
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(path); err != nil {
		return []object.Object{object.NewError(err.Error())}, nil
	}
	return []object.Object{object.Null}, nil
}

func nativeProcessPID(_ []object.Object) ([]object.Object, error) {
	return []object.Object{&object.Integer{Value: int64(os.Getpid())}}, nil
}

func nativeProcessParentPID(_ []object.Object) ([]object.Object, error) {
	return []object.Object{&object.Integer{Value: int64(os.Getppid())}}, nil
}

func makeProcessResult(stdout, stderr string, exitCode int64, success bool) *object.Dict {
	result := object.NewDict()
	result.Set(&object.String{Value: "stdout"}, &object.String{Value: stdout})
	result.Set(&object.String{Value: "stderr"}, &object.String{Value: stderr})
	result.Set(&object.String{Value: "exitCode"}, &object.Integer{Value: exitCode})
	result.Set(&object.String{Value: "success"}, object.BoolOf(success))
	return result
}

func nativeProcessExecute(args []object.Object) ([]object.Object, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("execute 至少需要 2 个参数")
	}
	command, err := requireProcessString("execute", args, 0)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(command) == "" {
		return []object.Object{makeProcessResult("", "", -1, false), object.NewError("命令不能为空")}, nil
	}
	argList, ok := args[1].(*object.Array)
	if !ok {
		return nil, fmt.Errorf("execute 第 2 个参数必须是数组")
	}
	cmdArgs := make([]string, 0, len(argList.Elements))
	for _, arg := range argList.Elements {
		text, ok := arg.(*object.String)
		if !ok {
			return nil, fmt.Errorf("命令参数必须全部是字符串")
		}
		cmdArgs = append(cmdArgs, text.Value)
	}
	var options object.Object = object.Null
	if len(args) >= 3 {
		options = args[2]
	}
	cmd := exec.Command(command, cmdArgs...)
	if cwd, _, err := optionString(options, "cwd"); err != nil {
		return nil, err
	} else if cwd != "" {
		cmd.Dir = cwd
	}
	if input, exists, err := optionString(options, "input"); err != nil {
		return nil, err
	} else if exists {
		cmd.Stdin = strings.NewReader(input)
	}
	if envOptions, exists := func() (*object.Dict, bool) {
		if options == nil || options == object.Null {
			return nil, false
		}
		dict, ok := options.(*object.Dict)
		return dict, ok
	}(); exists {
		value, exists := envOptions.Get(&object.String{Value: "env"})
		if exists && value != object.Null {
			envDict, ok := value.(*object.Dict)
			if !ok {
				return nil, fmt.Errorf("选项 env 必须是字典")
			}
			cmd.Env = os.Environ()
			for _, key := range envDict.Keys {
				name, ok := key.(*object.String)
				if !ok {
					return nil, fmt.Errorf("环境变量名必须是字符串")
				}
				envValue, ok := envDict.Get(key)
				if !ok {
					return nil, fmt.Errorf("环境变量 %s 缺少值", name.Value)
				}
				text, ok := envValue.(*object.String)
				if !ok {
					return nil, fmt.Errorf("环境变量 %s 的值必须是字符串", name.Value)
				}
				cmd.Env = append(cmd.Env, name.Value+"="+text.Value)
			}
		}
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if runErr == nil {
		return []object.Object{makeProcessResult(stdout.String(), stderr.String(), 0, true), object.Null}, nil
	}
	if exitErr, ok := runErr.(*exec.ExitError); ok {
		return []object.Object{makeProcessResult(stdout.String(), stderr.String(), int64(exitErr.ExitCode()), false), object.Null}, nil
	}
	return []object.Object{
		makeProcessResult(stdout.String(), stderr.String(), -1, false),
		object.NewError("无法执行命令：" + runErr.Error()),
	}, nil
}

func init() {
	RegisterNativeFunction("processArgs", nativeProcessArgs)
	RegisterNativeFunction("processScript", nativeProcessScript)
	RegisterNativeFunction("processEnv", nativeProcessEnv)
	RegisterNativeFunction("processSetEnv", nativeProcessSetEnv)
	RegisterNativeFunction("processWorkingDir", nativeProcessWorkingDir)
	RegisterNativeFunction("processChangeDir", nativeProcessChangeDir)
	RegisterNativeFunction("processPID", nativeProcessPID)
	RegisterNativeFunction("processParentPID", nativeProcessParentPID)
	RegisterNativeFunction("processExecute", nativeProcessExecute)
}
