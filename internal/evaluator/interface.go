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
					if reason := signatureMismatch(conflict, methodInfo{
						arity:       method.Arity,
						params:      method.Params,
						returnTypes: method.ReturnTypes,
					}, env); reason != "" {
						return fmt.Errorf("接口 %s 中方法 %s 的签名冲突（%s）", iface.Name, name, reason)
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

type methodInfo struct {
	arity       int
	params      []ast.Parameter
	returnTypes []ast.DeclType
	ambiguous   bool
}

func methodInfoOf(fn *Function) methodInfo {
	params := fn.Params
	if len(params) > 0 && params[0].Name == "自己" {
		params = params[1:]
	}
	return methodInfo{
		arity:       len(params),
		params:      append([]ast.Parameter(nil), params...),
		returnTypes: append([]ast.DeclType(nil), fn.ReturnTypes...),
	}
}

// ifaceMismatch 描述接口方法与实际方法签名的不匹配原因。
type ifaceMismatch struct {
	method    string
	reason    string
	ambiguous bool
}

// checkInterfaceSatisfied 判断值是否实现了接口的全部方法（含参数与返回类型）。
// 空 值视为可赋给接口（与 Go 的 nil 接口一致）。
func checkInterfaceSatisfied(iface *Interface, value object.Object, env *Environment) *ifaceMismatch {
	if value == nil || value.Kind() == object.KindNil {
		return nil
	}
	var methods map[string]methodInfo
	switch v := value.(type) {
	case *Instance:
		methods = classMethodSet(v.Class, map[string]bool{})
	case *object.Named:
		methods = namedMethodSet(v, env)
	default:
		methods = primitiveMethodSet(value, env)
	}
	for _, name := range iface.MethodNames() {
		want := iface.Methods[name]
		got, exists := methods[name]
		if !exists {
			return &ifaceMismatch{method: name, reason: "缺少"}
		}
		if got.ambiguous {
			return &ifaceMismatch{method: name, ambiguous: true}
		}
		if reason := signatureMismatch(want, got, env); reason != "" {
			return &ifaceMismatch{method: name, reason: reason}
		}
	}
	return nil
}

// signatureMismatch 比较接口方法签名与实现方法签名；匹配返回空串。
// 接口中未标注的类型（TypeAny）表示不约束；实现方法未标注类型同样跳过该项检查。
func signatureMismatch(want *InterfaceMethod, got methodInfo, env *Environment) string {
	wantParams := want.Params
	gotParams := got.params
	// 实现方法完全未标注类型（参数与返回均为 Any）时，跳过签名细查（鸭子类型）。
	implFullyUntyped := true
	for _, gp := range gotParams {
		if gp.Type != ast.TypeAny {
			implFullyUntyped = false
			break
		}
	}
	for _, rt := range got.returnTypes {
		if rt != ast.TypeAny {
			implFullyUntyped = false
		}
	}
	if implFullyUntyped {
		if len(gotParams) > 0 && gotParams[len(gotParams)-1].Variadic {
			fixed := len(gotParams) - 1
			if len(wantParams) < fixed {
				return fmt.Sprintf("参数数量不符：需要 %d 个，实际 %d 个", len(wantParams), got.arity)
			}
		} else if len(wantParams) != len(gotParams) {
			return fmt.Sprintf("参数数量不符：需要 %d 个，实际 %d 个", len(wantParams), got.arity)
		}
		return ""
	}
	// 可变参数：实现方法最后一个参数可变时，允许接口参数数量不小于固定参数数。
	if len(gotParams) > 0 && gotParams[len(gotParams)-1].Variadic {
		fixed := len(gotParams) - 1
		if len(wantParams) < fixed {
			return fmt.Sprintf("参数数量不符：需要 %d 个，实际 %d 个", len(wantParams), got.arity)
		}
	} else if len(wantParams) != len(gotParams) {
		return fmt.Sprintf("参数数量不符：需要 %d 个，实际 %d 个", len(wantParams), got.arity)
	}
	for i, wp := range wantParams {
		if wp.Type == ast.TypeAny {
			continue
		}
		var gp ast.Parameter
		if i < len(gotParams) {
			gp = gotParams[i]
		} else if len(gotParams) > 0 && gotParams[len(gotParams)-1].Variadic {
			gp = gotParams[len(gotParams)-1]
		}
		if gp.Type == ast.TypeAny {
			continue
		}
		if !signatureTypeCompatible(wp.Type, gp.Type) {
			return fmt.Sprintf("参数 %s 类型不符：需要 %s，实际 %s", wp.Name, wp.Type, gp.Type)
		}
	}
	if len(want.ReturnTypes) == 0 {
		return ""
	}
	if len(want.ReturnTypes) != len(got.returnTypes) {
		return fmt.Sprintf("返回值数量不符：需要 %d 个，实际 %d 个", len(want.ReturnTypes), len(got.returnTypes))
	}
	for i, wantType := range want.ReturnTypes {
		if wantType == ast.TypeAny {
			continue
		}
		gotType := got.returnTypes[i]
		if gotType == ast.TypeAny {
			continue
		}
		if !signatureTypeCompatible(wantType, gotType) {
			return fmt.Sprintf("返回值类型不符：需要 %s，实际 %s", wantType, gotType)
		}
	}
	return ""
}

// signatureTypeCompatible 比较接口签名与实现签名中的类型名是否一致。
// 凡语言采用 Go 的名义匹配：方法签名类型必须同名，不做协变/逆变。
func signatureTypeCompatible(want ast.DeclType, got ast.DeclType) bool {
	return want == got
}

// valueIsType 判断值是否属于/兼容给定类型名。支持：
// 基本类型（整数/小数/字符串/布尔/数组/字典/字节/字符）、模型（含嵌入链）、
// 接口（方法集满足）。空 仅兼容类型 空/错误。
func valueIsType(value object.Object, typeName string, env *Environment) bool {
	if value == nil {
		value = object.Null
	}
	if kind, ok := primitiveKindFromName(typeName); ok {
		if named, isNamed := value.(*object.Named); isNamed {
			return runtimeTypeName(named.Value) == kind
		}
		return runtimeTypeName(value) == kind
	}
	if value.Kind() == object.KindNil {
		switch typeName {
		case "空", "错误":
			return true
		}
		// 空 可赋给任何接口；名义模型/命名类型不匹配。
		if iface, ok := env.Runtime().lookupInterface(typeName, env); ok {
			return checkInterfaceSatisfied(iface, value, env) == nil
		}
		return false
	}
	// 自定义命名类型：身份严格一致。
	if named, ok := value.(*object.Named); ok {
		if named.TypeName == typeName {
			return true
		}
	}
	if _, isNamed := env.Runtime().namedTypes[typeName]; isNamed {
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
		return checkInterfaceSatisfied(iface, value, env) == nil
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
	// 目标是自定义命名类型：同身份成功；否则尝试按底层类型转换并包装。
	if namedType, ok := env.Runtime().namedTypes[expr.TypeName]; ok {
		if named, isNamedVal := value.(*object.Named); isNamedVal && named.TypeName == expr.TypeName {
			return &object.Tuple{Values: []object.Object{value, object.True}}, nil
		}
		if converted, err := convertToNamed(namedType, value, expr.Position); err == nil {
			return &object.Tuple{Values: []object.Object{converted, object.True}}, nil
		}
		return &object.Tuple{Values: []object.Object{object.Null, object.False}}, nil
	}
	if valueIsType(value, expr.TypeName, env) {
		return &object.Tuple{Values: []object.Object{value, object.True}}, nil
	}
	// 基本类型按 Go 风格做值转换（不抛错，失败返回 空、假）。
	if kind, ok := primitiveKindFromName(expr.TypeName); ok {
		// 命名类型值先解包到底层值再转换。
		if named, isNamed := value.(*object.Named); isNamed {
			value = named.Value
		}
		if converted, ok := convertToPrimitiveKind(value, kind); ok {
			return &object.Tuple{Values: []object.Object{converted, object.True}}, nil
		}
	}
	return &object.Tuple{Values: []object.Object{object.Null, object.False}}, nil
}

// convertToPrimitiveKind 在类型断言场景做基本类型值转换。
func convertToPrimitiveKind(value object.Object, kind string) (object.Object, bool) {
	if kind == "" {
		return nil, false
	}
	if runtimeTypeName(value) == kind {
		return value, true
	}
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
		out[name] = methodInfoOf(fn)
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

// namedMethodSet 收集自定义命名类型上定义的方法（接收者不计入参数数量）。
func namedMethodSet(value *object.Named, env *Environment) map[string]methodInfo {
	out := map[string]methodInfo{}
	namedType, ok := env.Runtime().namedTypes[value.TypeName]
	if !ok {
		return out
	}
	for name, fn := range namedType.Methods {
		info := methodInfoOf(fn)
		out[name] = info
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
		info := methodInfoOf(fn)
		out[name] = info
	}
	return out
}
