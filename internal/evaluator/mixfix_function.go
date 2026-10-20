package evaluator

import (
	"fmt"
	"strings"

	"fan/internal/ast"
	"fan/internal/object"
)

type mixfixFunctionRegistry struct {
	roots map[string]map[string]*Function
}

func newMixfixFunctionRegistry() *mixfixFunctionRegistry {
	return &mixfixFunctionRegistry{roots: map[string]map[string]*Function{}}
}

func (r *mixfixFunctionRegistry) register(segments []string, signature string, fn *Function) error {
	root := segments[0]
	if r.roots[root] == nil {
		r.roots[root] = map[string]*Function{}
	}
	if _, exists := r.roots[root][signature]; exists {
		return fmt.Errorf("重复的分段函数签名：%s", signature)
	}
	r.roots[root][signature] = fn
	return nil
}

func (r *mixfixFunctionRegistry) hasRoot(root string) bool {
	_, ok := r.roots[root]
	return ok
}

type mixfixCallSegment struct {
	connector string
	name      string
	size      int
	args      []ast.Expression
}

type mixfixCallChain struct {
	root       string
	moduleRoot string
	segments   []string
	connectors []string
	groupSize  []int
	arguments  [][]ast.Expression
}

func (r *mixfixFunctionRegistry) lookup(root string, signature string) (*Function, bool) {
	fn, ok := r.roots[root][signature]
	return fn, ok
}

// lookupModule 仅在指定模块中查找公开的分段函数。
func (r *mixfixFunctionRegistry) lookupModule(moduleName, root, signature string) (*Function, bool) {
	fn, ok := r.roots[root][signature]
	if !ok {
		return nil, false
	}
	if fn.ModuleName != moduleName || !fn.Public {
		return nil, false
	}
	return fn, true
}

// collectMixfixCallChain 识别分段函数调用链。env 用于区分“模块.根段(…)”
// 这种跨模块调用与普通的对象方法链。
func collectMixfixCallChain(node ast.Expression, env *Environment) (*mixfixCallChain, bool) {
	current := node
	var outward []mixfixCallSegment
	var rootArgs []ast.Expression
	moduleRoot := ""
rootLoop:
	for {
		switch call := current.(type) {
		case *ast.CallExpr:
			if rootIdent, isRootIdent := call.Callee.(*ast.Identifier); isRootIdent {
				rootArgs = call.Args
				current = rootIdent
				break rootLoop
			}
			member, isMember := call.Callee.(*ast.MemberExpr)
			if !isMember {
				return nil, false
			}
			if member.Spaced {
				return nil, false
			}
			// 链的最内层：模块.根段(参数)
			if rootIdent, ok := member.Object.(*ast.Identifier); ok && env.isModuleName(rootIdent.Name) {
				moduleRoot = rootIdent.Name
				rootArgs = call.Args
				current = &ast.Identifier{Name: member.Name}
				break rootLoop
			}
			outward = append(outward, mixfixCallSegment{
				connector: member.Connector,
				name:      member.Name,
				size:      len(call.Args),
				args:      call.Args,
			})
			current = member.Object
		case *ast.ImplicitReceiverCallExpr:
			if call.Spaced {
				return nil, false
			}
			outward = append(outward, mixfixCallSegment{
				connector: call.Connector,
				name:      call.Name,
				size:      len(call.Args),
				args:      call.Args,
			})
			current = call.Receiver
		case *ast.MemberExpr:
			// 无括号末段：仅允许出现在链尾，且必须紧邻。
			if call.Spaced {
				return nil, false
			}
			outward = append(outward, mixfixCallSegment{
				connector: call.Connector,
				name:      call.Name,
				size:      0,
			})
			current = call.Object
		default:
			return nil, false
		}
	}
	rootIdent, ok := current.(*ast.Identifier)
	if !ok || len(outward) == 0 {
		return nil, false
	}
	// “通过 模块 根(…)的段(…)”：裸根在模块作用域内解析。
	if moduleRoot == "" {
		if mod := env.throughModuleValue(); mod != nil {
			moduleRoot = mod.Name
		}
	}
	chain := &mixfixCallChain{
		root:       rootIdent.Name,
		moduleRoot: moduleRoot,
		segments:   []string{rootIdent.Name},
		connectors: []string{""},
		groupSize:  []int{len(rootArgs)},
		arguments:  [][]ast.Expression{rootArgs},
	}
	for i := len(outward) - 1; i >= 0; i-- {
		segment := outward[i]
		chain.connectors = append(chain.connectors, segment.connector)
		chain.segments = append(chain.segments, segment.name)
		chain.groupSize = append(chain.groupSize, segment.size)
		chain.arguments = append(chain.arguments, segment.args)
	}
	for i, size := range chain.groupSize {
		if size == 0 && i != len(chain.groupSize)-1 {
			return nil, false
		}
	}
	return chain, true
}

func evalMixfixCall(chain *mixfixCallChain, env *Environment, pos ast.Position) (object.Object, error) {
	var values []object.Object
	for _, group := range chain.arguments {
		groupValues := make([]object.Object, 0, len(group))
		for _, argExpr := range group {
			value, err := Eval(argExpr, env)
			if err != nil {
				return nil, err
			}
			if value == nil {
				value = object.Null
			}
			groupValues = append(groupValues, value)
		}
		values = append(values, flattenSpreadMarkers(spreadTuples(groupValues))...)
	}
	signature := buildRuntimeMixfixSignature(chain.segments, chain.connectors, chain.groupSize, values)
	var fn *Function
	var found bool
	if scopedMod := env.throughModuleValue(); scopedMod != nil && chain.moduleRoot == scopedMod.Name {
		// 通过 模块 根(…)的段(…)：在作用域模块的公开分段函数中匹配。
		fn, found = env.mixfixFunctions().lookupModule(scopedMod.Name, chain.root, signature)
	} else if chain.moduleRoot != "" {
		mod, err := env.getModule(chain.moduleRoot)
		if err != nil {
			return nil, &EvalError{Pos: pos, Reason: err.Error()}
		}
		fn, found = env.mixfixFunctions().lookupModule(mod.Name, chain.root, signature)
	} else {
		fn, found = env.mixfixFunctions().lookup(chain.root, signature)
	}
	if !found {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("找不到匹配的分段函数：%s", signature)}
	}
	return applyFunction(fn, values, pos)
}

func mixfixTypeLabel(value object.Object) string {
	switch value.(type) {
	case *object.Integer:
		return string(ast.TypeInt)
	case *object.Float:
		return string(ast.TypeFloat)
	case *object.String:
		return string(ast.TypeString)
	case *object.Bool:
		return string(ast.TypeBool)
	case *object.Array:
		return string(ast.TypeArray)
	case *object.Dict:
		return string(ast.TypeDict)
	case *Instance:
		return value.(*Instance).Class.Name
	}
	return runtimeTypeName(value)
}

func buildRuntimeMixfixSignature(segments []string, connectors []string, groupSizes []int, values []object.Object) string {
	var builder strings.Builder
	index := 0
	for segmentIndex, segment := range segments {
		builder.WriteString(connectors[segmentIndex])
		builder.WriteString(segment)
		builder.WriteString("(")
		for offset := 0; offset < groupSizes[segmentIndex]; offset++ {
			if offset > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(mixfixTypeLabel(values[index+offset]))
		}
		builder.WriteString(")")
		index += groupSizes[segmentIndex]
	}
	return builder.String()
}
