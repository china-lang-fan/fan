package analysis

import (
	"fan/internal/ast"
)

// checkCompositeLiterals 静态检查模型复合字面量：模型名是否存在、
// 字段是否属于该模型（直接字段或直接嵌入名）、字段是否重复。
// 仅在模型可静态定位时报告，避免对鸭子类型/动态写法误伤。
func (idx *Index) checkCompositeLiterals() []Diagnostic {
	infos := idx.buildClassInfo()
	var diags []Diagnostic
	walk(idx.root.Prog, func(node ast.Node) bool {
		lit, ok := node.(*ast.CompositeLiteral)
		if !ok {
			return true
		}
		modelName := compositeModelName(lit.Model)
		if modelName == "" {
			return true
		}
		info := infos[modelName]
		if info == nil {
			// 跨模块成员（工具.矩形）若索引未加载模块，保守跳过，避免误报。
			if _, isMember := lit.Model.(*ast.MemberExpr); isMember {
				return true
			}
			diags = append(diags, Diagnostic{
				Line:     lit.Position.Line,
				Column:   lit.Position.Column,
				EndLine:  lit.Position.Line,
				EndCol:   lit.Position.Column + len([]rune(modelName)),
				Severity: SeverityError,
				Code:     "composite-unknown-model",
				Message:  "未定义的模型：" + modelName,
			})
			return true
		}
		seen := map[string]bool{}
		for _, f := range lit.Fields {
			if seen[f.Name] {
				diags = append(diags, Diagnostic{
					Line:     f.NamePos.Line,
					Column:   f.NamePos.Column,
					EndLine:  f.NamePos.Line,
					EndCol:   f.NamePos.Column + len([]rune(f.Name)),
					Severity: SeverityError,
					Code:     "composite-duplicate-field",
					Message:  "复合字面量中字段 " + f.Name + " 重复指定",
				})
			}
			seen[f.Name] = true
			if info.fields[f.Name] {
				continue
			}
			isEmbed := false
			for _, e := range info.embeds {
				if e == f.Name {
					isEmbed = true
					break
				}
			}
			if isEmbed {
				continue
			}
			diags = append(diags, Diagnostic{
				Line:     f.NamePos.Line,
				Column:   f.NamePos.Column,
				EndLine:  f.NamePos.Line,
				EndCol:   f.NamePos.Column + len([]rune(f.Name)),
				Severity: SeverityError,
				Code:     "composite-unknown-field",
				Message:  "模型 " + modelName + " 没有字段 " + f.Name,
			})
		}
		return true
	})
	return diags
}

// compositeModelName 仅对可静态定位的模型名表达式返回名字：
// 标识符或模块成员（工具.矩形）。
func compositeModelName(model ast.Expression) string {
	switch n := model.(type) {
	case *ast.Identifier:
		return n.Name
	case *ast.MemberExpr:
		return n.Name
	}
	return ""
}
