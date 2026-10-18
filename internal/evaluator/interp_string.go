package evaluator

import (
	"strings"

	"fan/internal/ast"
	"fan/internal/object"
)

// evalInterpString 求值插值字符串：Parts[i] 与 Exprs[i] 交错拼接。
// 每个插值表达式求值后用其文本表示（Inspect）嵌入，空值显示为“空”。
func evalInterpString(node *ast.InterpString, env *Environment) (object.Object, error) {
	var b strings.Builder
	for i, part := range node.Parts {
		b.WriteString(part)
		if i < len(node.Exprs) {
			value, err := Eval(node.Exprs[i], env)
			if err != nil {
				return nil, err
			}
			if value == nil || value == object.Null {
				b.WriteString("空")
			} else {
				b.WriteString(value.Inspect())
			}
		}
	}
	return &object.String{Value: b.String()}, nil
}
