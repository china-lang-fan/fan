package evaluator

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

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

// valueIsType 判断值是否属于/兼容给定类型名。支持：
// 基本类型（整数/小数/字符串/布尔/数组/字典/字节/字符）、模型（含嵌入链）、
// 接口（方法集满足）。空 仅兼容类型 空/错误。
func valueIsType(value object.Object, typeName string, env *Environment) bool {
	if value == nil {
		value = object.Null
	}
	if kind, ok := primitiveKindFromName(typeName); ok {
		return runtimeTypeName(value) == kind
	}
	if value.Kind() == object.KindNil {
		switch typeName {
		case "空", "错误":
			return true
		}
		// 空 可赋给任何接口；名义模型类型不匹配。
		if iface, ok := env.Runtime().lookupInterface(typeName, env); ok {
			_, _, satisfied := checkInterfaceSatisfied(iface, value, env)
			return satisfied
		}
		return false
	}
	if inst, ok := value.(*Instance); ok {
		if inst.Class.Name == typeName {
			return true
		}
		if classHasEmbed(inst.Class, typeName, map[string]bool{}) {
			return true
		}
	}
	if iface, ok := env.Runtime().lookupInterface(typeName, env); ok {
		_, _, satisfied := checkInterfaceSatisfied(iface, value, env)
		return satisfied
	}
	return false
}

// classHasEmbed 判断模型的嵌入链中是否包含指定模型名。
func classHasEmbed(cls *Class, name string, seen map[string]bool) bool {
	if cls == nil || seen[cls.Name] {
		return false
	}
	seen[cls.Name] = true
	for _, embed := range cls.Embeds {
		if embed.Name == name || classHasEmbed(embed, name, seen) {
			return true
		}
	}
	return false
}

func evalTypeTestExpr(expr *ast.TypeTestExpr, env *Environment) (object.Object, error) {
	value, err := Eval(expr.Object, env)
	if err != nil {
		return nil, err
	}
	if value == nil {
		value = object.Null
	}
	return boolObject(valueIsType(value, expr.TypeName, env)), nil
}

func boolObject(b bool) *object.Bool {
	if b {
		return object.True
	}
	return object.False
}

func evalTypeCastExpr(expr *ast.TypeCastExpr, env *Environment) (object.Object, error) {
	value, err := Eval(expr.Object, env)
	if err != nil {
		return nil, err
	}
	if value == nil {
		value = object.Null
	}
	if valueIsType(value, expr.TypeName, env) {
		return &object.Tuple{Values: []object.Object{value, object.True}}, nil
	}
	// 基本类型按 Go 风格做值转换（不抛错，失败返回 空、假）。
	if kind, ok := primitiveKindFromName(expr.TypeName); ok {
		if converted, ok := convertToPrimitiveKind(value, kind); ok {
			return &object.Tuple{Values: []object.Object{converted, object.True}}, nil
		}
	}
	return &object.Tuple{Values: []object.Object{object.Null, object.False}}, nil
}

// convertToPrimitiveKind 在类型断言场景做基本类型值转换。
func convertToPrimitiveKind(value object.Object, kind string) (object.Object, bool) {
	switch kind {
	case "integer":
		switch v := value.(type) {
		case *object.Float:
			return &object.Integer{Value: int64(v.Value)}, true
		case *object.Bool:
			if v == object.True {
				return &object.Integer{Value: 1}, true
			}
			return &object.Integer{Value: 0}, true
		case *object.Byte:
			return &object.Integer{Value: int64(v.Value)}, true
		case *object.Char:
			return &object.Integer{Value: int64(v.Value)}, true
		case *object.String:
			if n, err := strconv.ParseInt(strings.TrimSpace(v.Value), 10, 64); err == nil {
				return &object.Integer{Value: n}, true
			}
		}
	case "float":
		switch v := value.(type) {
		case *object.Integer:
			return &object.Float{Value: float64(v.Value)}, true
		case *object.Bool:
			if v == object.True {
				return &object.Float{Value: 1}, true
			}
			return &object.Float{Value: 0}, true
		case *object.Byte:
			return &object.Float{Value: float64(v.Value)}, true
		case *object.Char:
			return &object.Float{Value: float64(v.Value)}, true
		case *object.String:
			if f, err := strconv.ParseFloat(strings.TrimSpace(v.Value), 64); err == nil {
				return &object.Float{Value: f}, true
			}
		}
	case "string":
		return &object.String{Value: value.Inspect()}, true
	case "byte":
		if n, ok := integerLike(value); ok && 0 <= n && n <= 255 {
			return &object.Byte{Value: byte(n)}, true
		}
	case "char":
		switch v := value.(type) {
		case *object.Char:
			return v, true
		case *object.String:
			runes := []rune(v.Value)
			if len(runes) == 1 {
				return &object.Char{Value: runes[0]}, true
			}
		case *object.Integer:
			return &object.Char{Value: rune(v.Value)}, true
		case *object.Byte:
			return &object.Char{Value: rune(v.Value)}, true
		}
	case "boolean":
		switch v := value.(type) {
		case *object.Integer:
			return boolObject(v.Value != 0), true
		case *object.Float:
			return boolObject(v.Value != 0), true
		case *object.String:
			switch strings.ToLower(strings.TrimSpace(v.Value)) {
			case "真", "true", "1", "yes", "on":
				return object.True, true
			case "假", "false", "0", "no", "off":
				return object.False, true
			}
		}
	}
	return nil, false
}

func integerLike(value object.Object) (int64, bool) {
	switch v := value.(type) {
	case *object.Integer:
		return v.Value, true
	case *object.Byte:
		return int64(v.Value), true
	case *object.Char:
		return int64(v.Value), true
	}
	return 0, false
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
