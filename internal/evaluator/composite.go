package evaluator

import (
	"fmt"

	"fan/internal/ast"
	"fan/internal/object"
)

// evalCompositeLiteral 求值模型复合字面量，如 矩形{宽: 3, 高: 4}。
// 语义与 Go 一致：创建零值实例后直接按字段名填充，不调用 初始化；
// 未列出的字段保持零值。嵌入字段只能整体赋值（人{动物: 动物{...}}）。
func evalCompositeLiteral(node *ast.CompositeLiteral, env *Environment) (object.Object, error) {
	cls, err := resolveModelClass(node.Model, env)
	if err != nil {
		return nil, err
	}
	if cls.Impl != nil {
		return nil, &EvalError{
			Pos:    node.Position,
			Reason: fmt.Sprintf("模型 %s 由标签自定义实现，不能用复合字面量构造，请使用 %s(...)", cls.Name, cls.Name),
		}
	}
	inst := newZeroInstance(cls)
	for _, f := range node.Fields {
		if err := assignCompositeField(cls, inst, f, env, node.Position); err != nil {
			return nil, &EvalError{Pos: f.NamePos, Reason: err.Error()}
		}
	}
	return inst, nil
}

// resolveModelClass 把模型名表达式解析为 *Class：
// 标识符在当前作用域查找，模块成员（工具.矩形）在模块导出中查找。
func resolveModelClass(model ast.Expression, env *Environment) (*Class, error) {
	switch n := model.(type) {
	case *ast.Identifier:
		val, ok := env.Get(n.Name)
		if !ok {
			return nil, fmt.Errorf("未定义的模型：%s", n.Name)
		}
		if cls, ok := val.(*Class); ok {
			return cls, nil
		}
		return nil, fmt.Errorf("%s 不是模型", n.Name)
	case *ast.MemberExpr:
		obj, err := Eval(n.Object, env)
		if err != nil {
			return nil, err
		}
		mod, ok := obj.(*object.Module)
		if !ok {
			return nil, fmt.Errorf("%s 不是模块，不能用复合字面量取模型", n.Object.String())
		}
		val, ok := mod.Exports[n.Name]
		if !ok {
			return nil, fmt.Errorf("模块 %s 没有导出 %s", mod.Name, n.Name)
		}
		if cls, ok := val.(*Class); ok {
			return cls, nil
		}
		return nil, fmt.Errorf("模块 %s 的 %s 不是模型", mod.Name, n.Name)
	}
	return nil, fmt.Errorf("复合字面量模型名非法")
}

func assignCompositeField(cls *Class, inst *Instance, f ast.CompositeField, env *Environment, pos ast.Position) error {
	// 直接字段。
	for _, decl := range cls.Fields {
		if decl.Name == f.Name {
			if !memberAccessible(decl.Visibility, cls, env, false) {
				return fmt.Errorf("模型 %s 的私有字段 %s 不能用复合字面量设置", cls.Name, f.Name)
			}
			val, err := Eval(f.Value, env)
			if err != nil {
				return err
			}
			inst.Fields[f.Name] = val
			return nil
		}
	}
	// 嵌入字段：只能整体赋一个同模型实例。
	for _, embed := range cls.Embeds {
		if embed.Name == f.Name {
			val, err := Eval(f.Value, env)
			if err != nil {
				return err
			}
			sub, ok := val.(*Instance)
			if !ok || sub.Class != embed {
				return fmt.Errorf("嵌入字段 %s 必须用同模型复合字面量整体赋值", f.Name)
			}
			if target := inst.embedByName(f.Name); target != nil {
				replaceEmbed(inst, f.Name, sub)
				return nil
			}
			return fmt.Errorf("模型 %s 没有嵌入 %s", cls.Name, f.Name)
		}
	}
	return fmt.Errorf("模型 %s 没有字段 %s", cls.Name, f.Name)
}

// replaceEmbed 用新实例替换同名嵌入子实例。
func replaceEmbed(inst *Instance, name string, sub *Instance) {
	for i, e := range inst.Embeds {
		if e.Class != nil && e.Class.Name == name {
			inst.Embeds[i] = sub
			return
		}
	}
}
