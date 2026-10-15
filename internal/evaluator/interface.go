package evaluator

import (
	"fmt"
	"sort"

	"fan/internal/ast"
	"fan/internal/object"
)

const KindInterface object.Kind = "接口"

// InterfaceMethod 是展平后的接口方法签名。
type InterfaceMethod struct {
	Name        string
	Arity       int
	Params      []ast.Parameter
	ReturnTypes []ast.DeclType
}

// Interface 是运行时的接口值。
type Interface struct {
	Name       string
	Methods    map[string]*InterfaceMethod
	EmbedNames []string
}

func (i *Interface) Kind() object.Kind { return KindInterface }
func (i *Interface) Inspect() string   { return fmt.Sprintf("<接口 %s>", i.Name) }

// MethodNames 返回排序后的方法名，便于稳定输出错误信息。
func (i *Interface) MethodNames() []string {
	names := make([]string, 0, len(i.Methods))
	for name := range i.Methods {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func evalInterfaceStmt(stmt *ast.InterfaceStmt, env *Environment) (object.Object, error) {
	methods := map[string]*InterfaceMethod{}
	for _, sig := range stmt.Methods {
		if _, exists := methods[sig.Name]; exists {
			return nil, &EvalError{Pos: sig.Position, Reason: fmt.Sprintf("接口 %s 的方法 %s 重复声明", stmt.Name, sig.Name)}
		}
		methods[sig.Name] = &InterfaceMethod{
			Name:        sig.Name,
			Arity:       len(sig.Params),
			Params:      sig.Params,
			ReturnTypes: sig.ReturnTypes,
		}
	}
	iface := &Interface{
		Name:       stmt.Name,
		Methods:    methods,
		EmbedNames: append([]string(nil), stmt.Embeds...),
	}
	rt := env.Runtime()
	if err := rt.registerInterface(iface, env); err != nil {
		return nil, &EvalError{Pos: stmt.Position, Reason: err.Error()}
	}
	if existing, ok := env.get(stmt.Name); ok {
		if old, ok := existing.(*Interface); ok {
			old.Methods = iface.Methods
			old.EmbedNames = iface.EmbedNames
			rt.interfaces[stmt.Name] = old
			return old, nil
		}
	}
	if err := env.declare(stmt.Name, iface, false, ast.TypeAny); err != nil {
		return nil, &EvalError{Pos: stmt.Position, Reason: err.Error()}
	}
	return iface, nil
}

func (r *Runtime) registerInterface(iface *Interface, env *Environment) error {
	// 展平嵌入接口，带环检测。
	visiting := map[string]bool{}
	visited := map[string]bool{iface.Name: true}
	var flatten func(i *Interface) error
	flatten = func(i *Interface) error {
		visiting[i.Name] = true
		for _, embedName := range i.EmbedNames {
			embed, ok := r.lookupInterface(embedName, env)
			if !ok {
				return fmt.Errorf("接口 %s 嵌入了未定义的接口 %s", i.Name, embedName)
			}
			if visiting[embedName] {
				return fmt.Errorf("检测到接口循环嵌入：%s", embedName)
			}
			if visited[embedName] {
				continue
			}
			visited[embedName] = true
			if err := flatten(embed); err != nil {
				return err
			}
			for name, method := range embed.Methods {
				if conflict, exists := iface.Methods[name]; exists {
					if conflict.Arity != method.Arity {
						return fmt.Errorf("接口 %s 中方法 %s 的签名冲突", iface.Name, name)
					}
					continue
				}
				iface.Methods[name] = method
			}
		}
		visiting[i.Name] = false
		return nil
	}
	if err := flatten(iface); err != nil {
		return err
	}
	r.interfaces[iface.Name] = iface
	return nil
}

func (r *Runtime) lookupInterface(name string, env *Environment) (*Interface, bool) {
	if iface, ok := r.interfaces[name]; ok {
		return iface, true
	}
	if env != nil {
		if v, ok := env.get(name); ok {
			if iface, ok := v.(*Interface); ok {
				r.interfaces[name] = iface
				return iface, true
			}
		}
	}
	return nil, false
}

// checkInterfaceSatisfied 判断值是否实现了接口的全部方法。
// 返回不满足时的方法名；ambiguous 为 true 表示该方法在嵌入链中有歧义。
// 空 值视为可赋给接口（与 Go 的 nil 接口一致）。
func checkInterfaceSatisfied(iface *Interface, value object.Object, env *Environment) (missing string, ambiguous bool, ok bool) {
	if value == nil || value.Kind() == object.KindNil {
		return "", false, true
	}
	var methods map[string]methodInfo
	switch v := value.(type) {
	case *Instance:
		methods = classMethodSet(v.Class, map[string]bool{})
	default:
		methods = primitiveMethodSet(value, env)
	}
	for _, name := range iface.MethodNames() {
		want := iface.Methods[name]
		got, exists := methods[name]
		if !exists {
			return name, false, false
		}
		if got.ambiguous {
			return name, true, false
		}
		if got.arity != want.Arity {
			return name, false, false
		}
	}
	return "", false, true
}

type methodInfo struct {
	arity     int
	ambiguous bool
}

// methodArity 计算模型方法对调用方可见的参数数量，显式 自己 接收者不计入。
func methodArity(fn *Function) int {
	arity := len(fn.Params)
	if arity > 0 && fn.Params[0].Name == "自己" {
		arity--
	}
	return arity
}

// classMethodSet 收集模型自身及嵌入提升后的方法，同层多个嵌入提供
// 同名方法时标记为 ambiguous（与 Class.lookupMethod 的歧义规则一致）。
func classMethodSet(cls *Class, seen map[string]bool) map[string]methodInfo {
	out := map[string]methodInfo{}
	if cls == nil || seen[cls.Name] {
		return out
	}
	seen[cls.Name] = true
	for name, fn := range cls.Methods {
		out[name] = methodInfo{arity: methodArity(fn)}
	}
	embedSets := make([]map[string]methodInfo, 0, len(cls.Embeds))
	for _, embed := range cls.Embeds {
		embedSets = append(embedSets, classMethodSet(embed, seen))
	}
	for name, info := range mergeAmbiguous(embedSets) {
		if _, exists := out[name]; exists {
			continue
		}
		out[name] = info
	}
	return out
}

// mergeAmbiguous 合并同一层多个嵌入的方法集；同名方法被两个及以上
// 嵌入提供时标记歧义。
func mergeAmbiguous(sets []map[string]methodInfo) map[string]methodInfo {
	out := map[string]methodInfo{}
	count := map[string]int{}
	for _, set := range sets {
		for name, info := range set {
			if info.ambiguous {
				out[name] = info
				count[name] = 2
				continue
			}
			if _, exists := out[name]; !exists {
				out[name] = info
			}
			count[name]++
		}
	}
	for name, n := range count {
		if n >= 2 {
			out[name] = methodInfo{ambiguous: true}
		}
	}
	return out
}

// primitiveMethodSet 通过原生方法注册表收集原生类型的方法。
func primitiveMethodSet(value object.Object, env *Environment) map[string]methodInfo {
	out := map[string]methodInfo{}
	registry := env.primitiveMethods()
	kind := runtimeTypeName(value)
	fns := registry.methods[kind]
	for name, fn := range fns {
		arity := len(fn.Params)
		if arity > 0 {
			arity-- // 去掉接收者参数
		}
		out[name] = methodInfo{arity: arity}
	}
	return out
}
