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
	store        map[string]*binding
	outer        *Environment
	BaseDir      string
	Loader       *Loader
	moduleExport bool
}

func NewEnvironment() *Environment {
	return &Environment{store: map[string]*binding{}}
}

func (e *Environment) SetBaseDir(dir string) {
	e.BaseDir = dir
}

func (e *Environment) SetLoader(l *Loader) {
	e.Loader = l
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	return &Environment{
		store:        map[string]*binding{},
		outer:        outer,
		BaseDir:      outer.BaseDir,
		Loader:       outer.Loader,
		moduleExport: outer.moduleExport,
	}
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
	if err := checkDeclType(dt, val); err != nil {
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
	if err := checkDeclType(b.declType, val); err != nil {
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

func checkDeclType(dt ast.DeclType, val object.Object) error {
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
	default:
		if val.Kind() == object.KindNil {
			return nil
		}
		if inst, ok := val.(*Instance); ok && inst.Class.Name == string(dt) {
			return nil
		}
		return fmt.Errorf("类型不匹配：声明为 %s，实际为 %s", dt, val.Kind())
	}
	if val.Kind() != want {
		return fmt.Errorf("类型不匹配：声明为 %s，实际为 %s", dt, val.Kind())
	}
	return nil
}
