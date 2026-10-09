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
	segments   []string
	connectors []string
	groupSize  []int
	arguments  [][]ast.Expression
}

func (r *mixfixFunctionRegistry) lookup(root string, signature string) (*Function, bool) {
	fn, ok := r.roots[root][signature]
	return fn, ok
}

func collectMixfixCallChain(node ast.Expression) (*mixfixCallChain, bool) {
	current := node
	var outward []mixfixCallSegment
	var rootArgs []ast.Expression
	for {
		switch call := current.(type) {
		case *ast.CallExpr:
			if rootIdent, isRootIdent := call.Callee.(*ast.Identifier); isRootIdent {
				rootArgs = call.Args
				current = rootIdent
				continue
			}
			member, isMember := call.Callee.(*ast.MemberExpr)
			if !isMember {
				return nil, false
			}
			outward = append(outward, mixfixCallSegment{
				connector: member.Connector,
				name:      member.Name,
				size:      len(call.Args),
				args:      call.Args,
			})
			current = member.Object
		case *ast.ImplicitReceiverCallExpr:
			outward = append(outward, mixfixCallSegment{
				connector: call.Connector,
				name:      call.Name,
				size:      len(call.Args),
				args:      call.Args,
			})
			current = call.Receiver
		default:
			goto root
		}
	}
root:
	rootIdent, ok := current.(*ast.Identifier)
	if !ok || len(outward) == 0 {
		return nil, false
	}
	chain := &mixfixCallChain{
		root:       rootIdent.Name,
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
	for _, size := range chain.groupSize {
		if size == 0 {
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
		values = append(values, spreadTuples(groupValues)...)
	}
	signature := buildRuntimeMixfixSignature(chain.segments, chain.connectors, chain.groupSize, values)
	fn, ok := env.mixfixFunctions().lookup(chain.root, signature)
	if !ok {
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
