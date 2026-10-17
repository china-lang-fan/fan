package evaluator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fan/internal/ast"
	"fan/internal/object"
	"fan/internal/parser"
)

type Loader struct {
	rt                *Runtime
	cache             map[string]*object.Module
	loading           map[string]bool
	modulesDirs       []string
	primitiveRegistry *primitiveMethodRegistry
	mixfixRegistry    *mixfixFunctionRegistry
}

func NewLoader(modulesDir string) *Loader {
	return &Loader{
		rt:          NewRuntime(),
		cache:       map[string]*object.Module{},
		loading:     map[string]bool{},
		modulesDirs: defaultModulesDirs(modulesDir),
	}
}

// NewLoaderForRuntime 创建绑定到指定运行实例的模块加载器。
func NewLoaderForRuntime(rt *Runtime, modulesDir string) *Loader {
	return &Loader{
		rt:          rt,
		cache:       map[string]*object.Module{},
		loading:     map[string]bool{},
		modulesDirs: defaultModulesDirs(modulesDir),
	}
}

func defaultModulesDirs(base string) []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(dir string) {
		if dir == "" {
			return
		}
		abs, err := filepath.Abs(dir)
		if err != nil || seen[abs] {
			return
		}
		seen[abs] = true
		dirs = append(dirs, abs)
	}
	add(base)
	if executable, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executable)
		add(executableDir)
		add(filepath.Join(executableDir, "sdk"))
		add(filepath.Join(executableDir, "..", "share", "fan", "sdk"))
		add(filepath.Join(executableDir, "..", "..", "share", "fan", "sdk"))
	}
	add("/usr/local/share/fan/sdk")
	add("/usr/share/fan/sdk")
	return dirs
}

func hasFanExtension(path string) bool {
	return strings.HasSuffix(path, ".fan") || strings.HasSuffix(path, ".凡")
}

func existingPath(path string) (string, bool) {
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	return abs, true
}

func firstExistingPath(paths ...string) (string, bool) {
	for _, path := range paths {
		if abs, ok := existingPath(path); ok {
			return abs, true
		}
	}
	return "", false
}

func (l *Loader) resolvePath(base string, path string) (string, error) {
	candidates := []string{path}
	if !hasFanExtension(path) {
		candidates = []string{path + ".凡", path + ".fan"}
	}
	if filepath.IsAbs(path) {
		if abs, ok := firstExistingPath(candidates...); ok {
			return abs, nil
		}
	} else {
		search := make([]string, 0, 2+len(l.modulesDirs))
		if base != "" {
			for _, candidate := range candidates {
				search = append(search, filepath.Join(base, candidate))
			}
		}
		for _, modulesDir := range l.modulesDirs {
			for _, candidate := range candidates {
				search = append(search, filepath.Join(modulesDir, candidate))
			}
		}
		search = append(search, candidates...)
		if abs, ok := firstExistingPath(search...); ok {
			return abs, nil
		}
	}
	return "", fmt.Errorf("找不到模块：%s", path)
}

func (l *Loader) Load(importerBase string, path string) (*object.Module, error) {
	full, err := l.resolvePath(importerBase, path)
	if err != nil {
		return nil, err
	}
	if mod, ok := l.cache[full]; ok {
		return mod, nil
	}
	if l.loading[full] {
		return nil, fmt.Errorf("检测到循环导入：%s", path)
	}
	l.loading[full] = true
	defer delete(l.loading, full)

	data, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	prog, errs := parser.ParseProgram(string(data))
	if len(errs) > 0 {
		return nil, fmt.Errorf("模块 %s 解析失败：%s", path, errs[0].Error())
	}

	base := filepath.Dir(full)
	registry := l.primitiveRegistry
	if registry == nil {
		registry = newPrimitiveMethodRegistry()
		l.primitiveRegistry = registry
	}
	mixfixRegistry := l.mixfixRegistry
	if mixfixRegistry == nil {
		mixfixRegistry = newMixfixFunctionRegistry()
		l.mixfixRegistry = mixfixRegistry
	}
	env := newEnvironment(l.rt)
	env.primitiveRegistry = registry
	env.mixfixRegistry = mixfixRegistry
	env.BaseDir = base
	env.SourceFile = full
	env.moduleName = moduleNameFromPath(full)
	env.Loader = l
	env.moduleExport = true

	if _, err := Eval(prog, env); err != nil {
		return nil, err
	}

	mod := object.NewModule(moduleNameFromPath(full))
	for name, b := range env.store {
		if b.visibility == ast.VisPublic {
			mod.Exports[name] = b.value
		}
	}
	l.cache[full] = mod
	return mod, nil
}

func moduleNameFromPath(full string) string {
	base := filepath.Base(full)
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	return base
}

func evalImportStmt(stmt *ast.ImportStmt, env *Environment) (object.Object, error) {
	if env.Loader == nil {
		env.SetLoader(NewLoaderForRuntime(env.Runtime(), env.BaseDir))
	}
	mod, err := env.Loader.Load(env.BaseDir, stmt.Path)
	if err != nil {
		return nil, &EvalError{Pos: stmt.Position, Reason: err.Error()}
	}
	if _, exists := env.find(stmt.Name); exists {
		return nil, &EvalError{Pos: stmt.Position, Reason: fmt.Sprintf("模块名 %s 已存在", stmt.Name)}
	}
	if err := env.declare(stmt.Name, mod, true, ast.TypeAny); err != nil {
		return nil, &EvalError{Pos: stmt.Position, Reason: err.Error()}
	}
	return mod, nil
}

// evalThroughExpr 在“通过 模块名 表达式”中以指定模块作用域求值表达式：
// 普通函数从模块公开成员查找；分段函数只匹配该模块公开的定义。
func evalThroughExpr(expr *ast.ThroughExpr, env *Environment) (object.Object, error) {
	mod, err := env.getModule(expr.ModuleName)
	if err != nil {
		return nil, &EvalError{Pos: expr.Position, Reason: err.Error()}
	}
	scoped := NewEnclosedEnvironment(env)
	scoped.throughModule = mod
	return Eval(expr.Expr, scoped)
}

func evalVisibilityStmt(stmt *ast.VisibilityStmt, env *Environment) (object.Object, error) {
	env.pendingVisibility = stmt.Visibility
	env.pendingVisSet = true
	res, err := Eval(stmt.Inner, env)
	if err != nil {
		return nil, err
	}
	env.pendingVisSet = false
	if b, ok := env.find(stmt.Name); ok {
		b.visibility = stmt.Visibility
	}
	return res, nil
}
