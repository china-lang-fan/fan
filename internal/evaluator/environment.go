package evaluator

import (
	"fmt"

	"fan/internal/ast"
	"fan/internal/object"
)

type binding struct {
	value    object.Object
	isConst  bool
	isExport bool
	declType ast.DeclType
}

type Environment struct {
	store             map[string]*binding
	outer             *Environment
	rt                *Runtime
	BaseDir           string
	Loader            *Loader
	moduleExport      bool
	primitiveRegistry *primitiveMethodRegistry
	mixfixRegistry    *mixfixFunctionRegistry
}

// NewEnvironment 创建一个绑定到新运行实例的根环境。
func NewEnvironment() *Environment {
	rt := NewRuntime()
	return newEnvironment(rt)
}

func newEnvironment(rt *Runtime) *Environment {
	return &Environment{
		store:             map[string]*binding{},
		rt:                rt,
		primitiveRegistry: newPrimitiveMethodRegistry(),
		mixfixRegistry:    newMixfixFunctionRegistry(),
	}
}

// NewEnvironmentForRuntime 创建绑定到指定运行实例的根环境。
func NewEnvironmentForRuntime(rt *Runtime) *Environment {
	return newEnvironment(rt)
}

// Runtime 返回环境所属的运行实例（沿外层链查找）。
func (e *Environment) Runtime() *Runtime {
	if e.rt != nil {
		return e.rt
	}
	if e.outer != nil {
		return e.outer.Runtime()
	}
	return nil
}

func (e *Environment) SetBaseDir(dir string) {
	e.BaseDir = dir
}

func (e *Environment) SetLoader(l *Loader) {
	e.Loader = l
	if l != nil {
		if l.rt == nil {
			l.rt = e.Runtime()
		} else if e.rt == nil {
			e.rt = l.rt
		}
		if l.primitiveRegistry == nil {
			l.primitiveRegistry = e.primitiveMethods()
		} else {
			e.primitiveRegistry = l.primitiveRegistry
		}
		if l.mixfixRegistry == nil {
			l.mixfixRegistry = e.mixfixFunctions()
		} else {
			e.mixfixRegistry = l.mixfixRegistry
		}
	}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	return &Environment{
		store:        map[string]*binding{},
		outer:        outer,
		rt:           outer.Runtime(),
		BaseDir:      outer.BaseDir,
		Loader:       outer.Loader,
		moduleExport: outer.moduleExport,
		mixfixRegistry: func() *mixfixFunctionRegistry {
			if outer.mixfixRegistry != nil {
				return outer.mixfixRegistry
			}
			if outer.outer != nil {
				return outer.outer.mixfixFunctions()
			}
			return newMixfixFunctionRegistry()
		}(),
	}
}

func (e *Environment) mixfixFunctions() *mixfixFunctionRegistry {
	if e.mixfixRegistry != nil {
		return e.mixfixRegistry
	}
	if e.outer != nil {
		return e.outer.mixfixFunctions()
	}
	e.mixfixRegistry = newMixfixFunctionRegistry()
	return e.mixfixRegistry
}

func (e *Environment) find(name string) (*binding, bool) {
	if b, ok := e.store[name]; ok {
		return b, true
	}
	if e.outer != nil {
		return e.outer.find(name)
	}
	return nil, false
}

func (e *Environment) declare(name string, val object.Object, isConst bool, dt ast.DeclType) error {
	if _, ok := e.store[name]; ok {
		return fmt.Errorf("变量 %s 已声明", name)
	}
	val, err := coerceDeclType(dt, val, e)
	if err != nil {
		return err
	}
	e.store[name] = &binding{value: val, isConst: isConst, declType: dt}
	return nil
}

func (e *Environment) assign(name string, val object.Object) error {
	b, ok := e.find(name)
	if !ok {
		return fmt.Errorf("变量 %s 未声明", name)
	}
	if b.isConst {
		return fmt.Errorf("常量 %s 不可再赋值", name)
	}
	val, err := coerceDeclType(b.declType, val, e)
	if err != nil {
		return err
	}
	b.value = val
	return nil
}

func (e *Environment) Get(name string) (object.Object, bool) {
	return e.get(name)
}

func (e *Environment) get(name string) (object.Object, bool) {
	b, ok := e.find(name)
	if !ok {
		return nil, false
	}
	return b.value, true
}

func coerceDeclType(dt ast.DeclType, val object.Object, env *Environment) (object.Object, error) {
	if dt == ast.TypeByte {
		if b, ok := val.(*object.Byte); ok {
			return b, nil
		}
		if n, ok := val.(*object.Integer); ok {
			if n.Value < 0 || n.Value > 255 {
				return nil, fmt.Errorf("字节超出范围：%d（允许 0–255）", n.Value)
			}
			return &object.Byte{Value: byte(n.Value)}, nil
		}
		return nil, fmt.Errorf("类型不匹配：声明为 字节，实际为 %s", val.Kind())
	}
	if err := checkDeclType(dt, val, env); err != nil {
		return nil, err
	}
	return val, nil
}

func checkDeclType(dt ast.DeclType, val object.Object, env *Environment) error {
	if dt == ast.TypeAny {
		return nil
	}
	want := object.Kind("")
	switch dt {
	case ast.TypeInt:
		want = object.KindInt
	case ast.TypeFloat:
		want = object.KindFloat
	case ast.TypeString:
		want = object.KindString
	case ast.TypeBool:
		want = object.KindBool
	case ast.TypeArray:
		want = object.KindArray
	case ast.TypeDict:
		want = object.KindDict
	case ast.TypeError:
		if val.Kind() == object.KindNil {
			return nil
		}
		want = object.KindError
	case ast.TypeByte:
		switch v := val.(type) {
		case *object.Byte:
			return nil
		case *object.Integer:
			if v.Value < 0 || v.Value > 255 {
				return fmt.Errorf("字节超出范围：%d（允许 0–255）", v.Value)
			}
			return nil
		}
		return fmt.Errorf("类型不匹配：声明为 字节，实际为 %s", val.Kind())
	case ast.TypeChar:
		if val.Kind() == object.KindChar {
			return nil
		}
		return fmt.Errorf("类型不匹配：声明为 字符，实际为 %s", val.Kind())
	default:
		if val.Kind() == object.KindNil {
			return nil
		}
		// 自定义命名类型：身份必须严格一致（参考 Go）。
		if named, ok := val.(*object.Named); ok && named.TypeName == string(dt) {
			return nil
		}
		if inst, ok := val.(*Instance); ok && inst.Class.Name == string(dt) {
			return nil
		}
		if env != nil {
			rt := env.Runtime()
			// 自定义命名类型：不允许隐式从底层类型赋值，需显式 类型名(值) 转换。
			if _, isNamed := rt.namedTypes[string(dt)]; isNamed {
				return fmt.Errorf("类型不匹配：声明为 %s，实际为 %s（需显式转换）", dt, val.Kind())
			}
			// 声明的类型名可能是接口：按方法集做结构化满足检查。
			if iface, ok := rt.lookupInterface(string(dt), env); ok {
				if name, ambiguous, satisfied := checkInterfaceSatisfied(iface, val, env); !satisfied {
					if ambiguous {
						return fmt.Errorf("类型不匹配：声明为 %s，实际为 %s（方法 %s 在嵌入链中有歧义）", dt, val.Kind(), name)
					}
					return fmt.Errorf("类型不匹配：声明为 %s，实际为 %s（缺少方法 %s）", dt, val.Kind(), name)
				}
				return nil
			}
		}
		return fmt.Errorf("类型不匹配：声明为 %s，实际为 %s", dt, val.Kind())
	}
	if val.Kind() != want {
		return fmt.Errorf("类型不匹配：声明为 %s，实际为 %s", dt, val.Kind())
	}
	return nil
}
