package evaluator

import (
	"fmt"
	"strings"

	"fan/internal/ast"
	"fan/internal/object"
)

type EvalError struct {
	Pos    ast.Position
	Reason string
	Stack  []object.StackFrame
}

func (e *EvalError) Error() string {
	if e.Pos.Line > 0 {
		return fmt.Sprintf("第%d行%d列：%s%s", e.Pos.Line, e.Pos.Column, e.Reason, formatStack(e.Stack))
	}
	return e.Reason + formatStack(e.Stack)
}

func formatStack(frames []object.StackFrame) string {
	if len(frames) == 0 {
		return ""
	}
	var b strings.Builder
	for _, f := range frames {
		b.WriteString("\n")
		b.WriteString(f.String())
	}
	return b.String()
}

func Eval(node ast.Node, env *Environment) (object.Object, error) {
	switch n := node.(type) {
	case *ast.Program:
		return evalProgram(n, env)
	case *ast.ExpressionStmt:
		return Eval(n.Expression, env)
	case *ast.VarDecl:
		return evalVarDecl(n, env)
	case *ast.MultiDecl:
		return evalMultiDecl(n, env)
	case *ast.MultiAssign:
		return evalMultiAssign(n, env)
	case *ast.CompoundAssignStmt:
		return evalCompoundAssign(n, env)
	case *ast.UpdateExpr:
		return evalUpdateExpr(n, env)
	case *ast.SwitchStmt:
		return evalSwitchStmt(n, env)
	case *ast.IfStmt:
		return evalIfStmt(n, env)
	case *ast.BlockStmt:
		return evalBlock(n, NewEnclosedEnvironment(env))
	case *ast.ArrayLiteral:
		return evalArrayLiteral(n, env)
	case *ast.DictLiteral:
		return evalDictLiteral(n, env)
	case *ast.IndexExpr:
		return evalIndexExpr(n, env)
	case *ast.IndexAssignStmt:
		return evalIndexAssign(n, env)
	case *ast.CallExpr:
		return evalCallExprDispatch(n, env)
	case *ast.FunctionLiteral:
		return evalFunctionLiteral(n, env)
	case *ast.ReturnStmt:
		return evalReturnStmt(n, env)
	case *ast.MemberExpr:
		return evalMemberExpression(n, env)
	case *ast.ImportStmt:
		return evalImportStmt(n, env)
	case *ast.VisibilityStmt:
		return evalVisibilityStmt(n, env)
	case *ast.ClassStmt:
		return evalClassStmt(n, env)
	case *ast.InterfaceStmt:
		return evalInterfaceStmt(n, env)
	case *ast.TypeDefStmt:
		return evalTypeDefStmt(n, env)
	case *ast.TypeTestExpr:
		return evalTypeTestExpr(n, env)
	case *ast.TypeCastExpr:
		return evalTypeCastExpr(n, env)
	case *ast.MethodDef:
		return evalMethodDef(n, env)
	case *ast.FieldAssignExpr:
		return evalFieldAssign(n, env)
	case *ast.WhileStmt:
		return evalWhileStmt(n, env)
	case *ast.RepeatStmt:
		return evalRepeatStmt(n, env)
	case *ast.TimesStmt:
		return evalTimesStmt(n, env)
	case *ast.ForEachStmt:
		return evalForEachStmt(n, env)
	case *ast.DeferStmt:
		return evalDeferStmt(env.Runtime(), n, env)
	case *ast.ThrowStmt:
		return evalThrowStmt(n, env)
	case *ast.CheckExpr:
		return evalCheckExpr(n, env)
	case *ast.BreakStmt:
		return nil, &controlSignal{kind: controlBreak, pos: n.Position}
	case *ast.ContinueStmt:
		return nil, &controlSignal{kind: controlContinue, pos: n.Position}
	case *ast.IntegerLiteral:
		return &object.Integer{Value: n.Value}, nil
	case *ast.FloatLiteral:
		return &object.Float{Value: n.Value}, nil
	case *ast.StringLiteral:
		return &object.String{Value: n.Value}, nil
	case *ast.CharLiteral:
		return &object.Char{Value: n.Value}, nil
	case *ast.BoolLiteral:
		return object.BoolOf(n.Value), nil
	case *ast.NilLiteral:
		return object.Null, nil
	case *ast.Identifier:
		return evalIdentifier(n, env)
	case *ast.MaybeCallExpr:
		return evalMaybeCallExpr(n, env)
	case *ast.TernaryExpr:
		return evalTernaryExpr(n, env)
	case *ast.BinaryExpr:
		return evalBinaryExpr(n, env)
	case *ast.ImplicitReceiverCallExpr:
		return evalImplicitReceiverCallExpr(n, env)
	case *ast.UnaryExpr:
		return evalUnaryExpr(n, env)
	}
	return nil, &EvalError{Reason: fmt.Sprintf("不支持求值的节点类型：%T", node)}
}

func evalProgram(prog *ast.Program, env *Environment) (object.Object, error) {
	rt := env.Runtime()
	frame := rt.pushFrame()
	defer rt.popFrame()
	var last object.Object
	for _, stmt := range prog.Statements {
		res, err := Eval(stmt, env)
		if err != nil {
			if sig, ok := err.(*controlSignal); ok {
				return nil, sig
			}
			if ret, ok := err.(*returnSignal); ok {
				if len(ret.values) == 1 {
					return ret.values[0], nil
				}
				return &object.Tuple{Values: ret.values}, nil
			}
			if panicErr, ok := err.(*panicSignal); ok {
				frame.panic = panicErr
				deferErr := runDefers(rt, frame)
				if deferErr != nil {
					return nil, deferErr
				}
				if frame.panic == nil {
					return object.Null, nil
				}
				return nil, frame.panic
			}
			deferErr := runDefers(rt, frame)
			if deferErr != nil {
				return nil, deferErr
			}
			pos := stmt.Pos()
			if evalErr, ok := err.(*EvalError); ok {
				if evalErr.Stack == nil {
					evalErr.Stack = rt.Stack()
				}
				return nil, evalErr
			}
			return nil, &EvalError{Pos: pos, Reason: err.Error(), Stack: rt.Stack()}
		}
		last = res
	}
	if err := runDefers(rt, frame); err != nil {
		return nil, err
	}
	return last, nil
}

func evalIfStmt(stmt *ast.IfStmt, env *Environment) (object.Object, error) {
	for _, br := range stmt.Branches {
		cond, err := Eval(br.Condition, env)
		if err != nil {
			return nil, err
		}
		if cond.Kind() != object.KindBool {
			return nil, &EvalError{
				Pos:    br.Condition.Pos(),
				Reason: fmt.Sprintf("如果 的条件必须是布尔，实际是 %s", cond.Kind()),
			}
		}
		if cond.(*object.Bool).Value {
			return evalBlock(br.Body, NewEnclosedEnvironment(env))
		}
	}
	if stmt.Else != nil {
		return evalBlock(stmt.Else, NewEnclosedEnvironment(env))
	}
	return object.Null, nil
}

func evalBlock(block *ast.BlockStmt, env *Environment) (object.Object, error) {
	var last object.Object = object.Null
	for _, stmt := range block.Statements {
		res, err := Eval(stmt, env)
		if err != nil {
			return nil, err
		}
		if res != nil {
			last = res
		}
	}
	return last, nil
}

func evalVarDecl(decl *ast.VarDecl, env *Environment) (object.Object, error) {
	val, err := Eval(decl.Value, env)
	if err != nil {
		return nil, err
	}
	if fn, ok := val.(*Function); ok && len(fn.NameSegments) > 1 {
		root := fn.NameSegments[0]
		if _, exists := env.find(root); exists {
			return nil, &EvalError{Pos: decl.Position, Reason: fmt.Sprintf("分段函数首段名 %s 已被占用", root)}
		}
		if err := env.mixfixFunctions().register(fn.NameSegments, fn.Signature, fn); err != nil {
			return nil, &EvalError{Pos: decl.Position, Reason: err.Error()}
		}
	}
	if decl.IsExplicit {
		if err := env.declare(decl.Name, val, decl.IsConst, decl.DeclType); err != nil {
			return nil, &EvalError{Pos: decl.Position, Reason: err.Error()}
		}
		return val, nil
	}
	if _, ok := env.find(decl.Name); ok {
		if err := env.assign(decl.Name, val); err != nil {
			return nil, &EvalError{Pos: decl.Position, Reason: err.Error()}
		}
		return val, nil
	}
	if err := env.declare(decl.Name, val, false, ast.TypeAny); err != nil {
		return nil, &EvalError{Pos: decl.Position, Reason: err.Error()}
	}
	return val, nil
}

func requireTuple(pos ast.Position, val object.Object, count int) ([]object.Object, error) {
	tuple, ok := val.(*object.Tuple)
	if !ok {
		if count == 1 {
			return []object.Object{val}, nil
		}
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("多目标赋值需要 %d 个返回值，实际是单个 %s", count, val.Kind())}
	}
	if len(tuple.Values) != count {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("多目标赋值数量不符：左侧 %d 个，右侧 %d 个", count, len(tuple.Values))}
	}
	return tuple.Values, nil
}

func evalMultiDecl(decl *ast.MultiDecl, env *Environment) (object.Object, error) {
	val, err := Eval(decl.Value, env)
	if err != nil {
		return nil, err
	}
	if val == nil {
		val = object.Null
	}
	values, err := requireTuple(decl.Position, val, len(decl.Names))
	if err != nil {
		return nil, err
	}
	for i, name := range decl.Names {
		if err := env.declare(name, values[i], decl.IsConst, ast.TypeAny); err != nil {
			return nil, &EvalError{Pos: decl.Position, Reason: err.Error()}
		}
	}
	return val, nil
}

func evalMultiAssign(stmt *ast.MultiAssign, env *Environment) (object.Object, error) {
	val, err := Eval(stmt.Value, env)
	if err != nil {
		return nil, err
	}
	if val == nil {
		val = object.Null
	}
	values, err := requireTuple(stmt.Position, val, len(stmt.Names))
	if err != nil {
		return nil, err
	}
	for _, name := range stmt.Names {
		if _, ok := env.find(name); !ok {
			return nil, &EvalError{Pos: stmt.Position, Reason: fmt.Sprintf("变量 %s 未声明", name)}
		}
	}
	for i, name := range stmt.Names {
		if err := env.assign(name, values[i]); err != nil {
			return nil, &EvalError{Pos: stmt.Position, Reason: err.Error()}
		}
	}
	return val, nil
}

func posPrefix(pos ast.Position) string {
	return fmt.Sprintf("第%d行", pos.Line)
}

var _ = posPrefix

func evalIdentifier(ident *ast.Identifier, env *Environment) (object.Object, error) {
	val, ok := env.get(ident.Name)
	if !ok {
		return nil, fmt.Errorf("变量 %s 未声明", ident.Name)
	}
	return val, nil
}

func evalMaybeCallExpr(expr *ast.MaybeCallExpr, env *Environment) (object.Object, error) {
	if chain, ok := collectMixfixCallChain(expr.Callee); ok && env.mixfixFunctions().hasRoot(chain.root) {
		return evalMixfixCall(chain, env, expr.Position)
	}
	if member, ok := expr.Callee.(*ast.MemberExpr); ok {
		target, err := Eval(member.Object, env)
		if err != nil {
			return nil, err
		}
		if !expr.AutoCall {
			return evalFieldAccess(member, env, target)
		}
		if fn, ok := env.primitiveMethods().lookup(target, member.Name); ok && len(fn.Params) == 1 {
			return applyFunction(fn, []object.Object{target}, expr.Position)
		}
		if inst, ok := target.(*Instance); ok {
			if fn := inst.Class.lookupMethod(member.Name); fn != nil && len(fn.Params) == 0 {
				return applyFunction(fn.bind(inst), nil, expr.Position)
			}
		}
		return evalFieldAccess(member, env, target)
	}
	val, err := Eval(expr.Callee, env)
	if err != nil {
		return nil, err
	}
	if expr.AutoCall {
		if fn, ok := val.(*Function); ok && len(fn.Params) == 0 {
			return applyFunction(fn, nil, expr.Position)
		}
	}
	return val, nil
}

func evalTernaryExpr(expr *ast.TernaryExpr, env *Environment) (object.Object, error) {
	cond, err := Eval(expr.Cond, env)
	if err != nil {
		return nil, err
	}
	b, ok := cond.(*object.Bool)
	if !ok {
		return nil, &EvalError{Pos: expr.Position, Reason: fmt.Sprintf("三元表达式条件必须是布尔，实际是 %s", cond.Kind())}
	}
	if b.Value {
		return Eval(expr.Then, env)
	}
	return Eval(expr.Else, env)
}

func evalBinaryExpr(expr *ast.BinaryExpr, env *Environment) (object.Object, error) {
	switch expr.Op {
	case "且", "&&":
		return evalAndExpr(expr, env)
	case "或", "||":
		return evalOrExpr(expr, env)
	}
	left, err := Eval(expr.Left, env)
	if err != nil {
		return nil, err
	}
	right, err := Eval(expr.Right, env)
	if err != nil {
		return nil, err
	}
	return evalBinaryOp(expr, left, right)
}

func evalAndExpr(expr *ast.BinaryExpr, env *Environment) (object.Object, error) {
	left, err := Eval(expr.Left, env)
	if err != nil {
		return nil, err
	}
	if left.Kind() != object.KindBool {
		return nil, &EvalError{Pos: expr.Position, Reason: fmt.Sprintf("且 左侧必须是布尔，实际是 %s", left.Kind())}
	}
	if !left.(*object.Bool).Value {
		return object.False, nil
	}
	right, err := Eval(expr.Right, env)
	if err != nil {
		return nil, err
	}
	if right.Kind() != object.KindBool {
		return nil, &EvalError{Pos: expr.Position, Reason: fmt.Sprintf("且 右侧必须是布尔，实际是 %s", right.Kind())}
	}
	return right, nil
}

func evalOrExpr(expr *ast.BinaryExpr, env *Environment) (object.Object, error) {
	left, err := Eval(expr.Left, env)
	if err != nil {
		return nil, err
	}
	if left.Kind() != object.KindBool {
		return nil, &EvalError{Pos: expr.Position, Reason: fmt.Sprintf("或 左侧必须是布尔，实际是 %s", left.Kind())}
	}
	if left.(*object.Bool).Value {
		return object.True, nil
	}
	right, err := Eval(expr.Right, env)
	if err != nil {
		return nil, err
	}
	if right.Kind() != object.KindBool {
		return nil, &EvalError{Pos: expr.Position, Reason: fmt.Sprintf("或 右侧必须是布尔，实际是 %s", right.Kind())}
	}
	return right, nil
}

func evalBinaryOp(expr *ast.BinaryExpr, left, right object.Object) (object.Object, error) {
	left = unwrapNamed(left)
	right = unwrapNamed(right)
	op := expr.Op
	pos := expr.Position

	switch op {
	case "加", "+":
		return evalAdd(pos, left, right)
	case "减", "-":
		return evalSub(pos, left, right)
	case "乘", "*":
		return evalMul(pos, left, right)
	case "除", "/":
		return evalDiv(pos, left, right)
	case "取余", "%":
		return evalMod(pos, left, right)
	case "等于", "==":
		return evalEq(pos, left, right)
	case "不等于", "!=":
		res, err := evalEq(pos, left, right)
		if err != nil {
			return nil, err
		}
		return object.BoolOf(!res.(*object.Bool).Value), nil
	case "小于", "<":
		return evalCompare(pos, left, right, -1)
	case "小于等于", "<=":
		return evalCompare(pos, left, right, 0, -1)
	case "大于", ">":
		return evalCompare(pos, left, right, 1)
	case "大于等于", ">=":
		return evalCompare(pos, left, right, 0, 1)
	case "且", "&&":
		return evalAnd(pos, left, right)
	case "或", "||":
		return evalOr(pos, left, right)
	case "位与", "&":
		return evalBitwiseBinary(pos, left, right, "位与")
	case "位或", "|":
		return evalBitwiseBinary(pos, left, right, "位或")
	case "异或", "^":
		return evalBitwiseBinary(pos, left, right, "异或")
	case "左移", "<<":
		return evalBitwiseBinary(pos, left, right, "左移")
	case "右移", ">>":
		return evalBitwiseBinary(pos, left, right, "右移")
	default:
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("未知运算符：%s", op)}
	}
}

func evalAdd(pos ast.Position, left, right object.Object) (object.Object, error) {
	if left.Kind() == object.KindArray && right.Kind() == object.KindArray {
		return evalArrayConcat(pos, left.(*object.Array), right.(*object.Array)), nil
	}
	if left.Kind() == object.KindString && right.Kind() == object.KindString {
		return &object.String{Value: left.(*object.String).Value + right.(*object.String).Value}, nil
	}
	// 字符串 + 字符 => 字符串
	if left.Kind() == object.KindString && right.Kind() == object.KindChar {
		return &object.String{Value: left.(*object.String).Value + string(right.(*object.Char).Value)}, nil
	}
	if left.Kind() == object.KindChar && right.Kind() == object.KindString {
		return &object.String{Value: string(left.(*object.Char).Value) + right.(*object.String).Value}, nil
	}
	// 字符 + 整数 => 字符
	if lc, ok := left.(*object.Char); ok {
		if rv, ok := intValue(right); ok {
			return &object.Char{Value: lc.Value + rune(rv)}, nil
		}
	}
	if rc, ok := right.(*object.Char); ok {
		if lv, ok := intValue(left); ok {
			return &object.Char{Value: rc.Value + rune(lv)}, nil
		}
	}
	if lv, ok := intValue(left); ok {
		if rv, ok := intValue(right); ok {
			return &object.Integer{Value: lv + rv}, nil
		}
	}
	lv := toFloat(left)
	rv := toFloat(right)
	if lv == nil || rv == nil {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("加法不支持 %s + %s", left.Kind(), right.Kind())}
	}
	return &object.Float{Value: *lv + *rv}, nil
}

func evalSub(pos ast.Position, left, right object.Object) (object.Object, error) {
	if lv, ok := intValue(left); ok {
		if rv, ok := intValue(right); ok {
			return &object.Integer{Value: lv - rv}, nil
		}
	}
	lv := toFloat(left)
	rv := toFloat(right)
	if lv == nil || rv == nil {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("减法不支持 %s - %s", left.Kind(), right.Kind())}
	}
	return &object.Float{Value: *lv - *rv}, nil
}

func evalMul(pos ast.Position, left, right object.Object) (object.Object, error) {
	if lv, ok := intValue(left); ok {
		if rv, ok := intValue(right); ok {
			return &object.Integer{Value: lv * rv}, nil
		}
	}
	lv := toFloat(left)
	rv := toFloat(right)
	if lv == nil || rv == nil {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("乘法不支持 %s * %s", left.Kind(), right.Kind())}
	}
	return &object.Float{Value: *lv * *rv}, nil
}

func evalDiv(pos ast.Position, left, right object.Object) (object.Object, error) {
	if lv, ok := intValue(left); ok {
		if rv, ok := intValue(right); ok {
			if rv == 0 {
				return nil, &EvalError{Pos: pos, Reason: "除数不能为零"}
			}
			return &object.Integer{Value: lv / rv}, nil
		}
	}
	lv := toFloat(left)
	rv := toFloat(right)
	if lv == nil || rv == nil {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("除法不支持 %s / %s", left.Kind(), right.Kind())}
	}
	if *rv == 0 {
		return nil, &EvalError{Pos: pos, Reason: "除数不能为零"}
	}
	return &object.Float{Value: *lv / *rv}, nil
}

func evalMod(pos ast.Position, left, right object.Object) (object.Object, error) {
	lv, lok := intValue(left)
	rv, rok := intValue(right)
	if !lok || !rok {
		return nil, &EvalError{Pos: pos, Reason: "取余仅支持整数 % 整数"}
	}
	if rv == 0 {
		return nil, &EvalError{Pos: pos, Reason: "除数不能为零"}
	}
	return &object.Integer{Value: lv % rv}, nil
}

func evalBitwiseBinary(pos ast.Position, left, right object.Object, op string) (object.Object, error) {
	lv, lok := intValue(left)
	rv, rok := intValue(right)
	if !lok || !rok {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("%s仅支持整数家族（整数/字节），实际 %s 与 %s", op, left.Kind(), right.Kind())}
	}
	var result int64
	switch op {
	case "位与":
		result = lv & rv
	case "位或":
		result = lv | rv
	case "异或":
		result = lv ^ rv
	case "左移":
		if rv < 0 {
			return nil, &EvalError{Pos: pos, Reason: "左移位数不能为负"}
		}
		if rv > 63 {
			result = 0
		} else {
			result = lv << uint(rv)
		}
	case "右移":
		if rv < 0 {
			return nil, &EvalError{Pos: pos, Reason: "右移位数不能为负"}
		}
		if rv > 63 {
			if lv < 0 {
				result = -1
			}
		} else {
			result = lv >> uint(rv)
		}
	default:
		return nil, &EvalError{Pos: pos, Reason: "未知位运算：" + op}
	}
	return &object.Integer{Value: result}, nil
}

func evalEq(pos ast.Position, left, right object.Object) (object.Object, error) {
	left = unwrapNamed(left)
	right = unwrapNamed(right)
	if left.Kind() == object.KindError && right.Kind() == object.KindNil {
		return object.False, nil
	}
	if left.Kind() == object.KindNil && right.Kind() == object.KindError {
		return object.False, nil
	}
	// 整数家族（整数/字节）与字符（可比较码点）等值比较
	if lv, lok := intValue(left); lok {
		if rv, rok := intValue(right); rok {
			return object.BoolOf(lv == rv), nil
		}
		if rc, ok := right.(*object.Char); ok {
			return object.BoolOf(lv == int64(rc.Value)), nil
		}
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("不同类型不可比较：%s 与 %s", left.Kind(), right.Kind())}
	}
	if lc, ok := left.(*object.Char); ok {
		if rv, rok := intValue(right); rok {
			return object.BoolOf(int64(lc.Value) == rv), nil
		}
	}
	if left.Kind() != right.Kind() {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("不同类型不可比较：%s 与 %s", left.Kind(), right.Kind())}
	}
	switch left.Kind() {
	case object.KindInt:
		return object.BoolOf(left.(*object.Integer).Value == right.(*object.Integer).Value), nil
	case object.KindFloat:
		return object.BoolOf(left.(*object.Float).Value == right.(*object.Float).Value), nil
	case object.KindString:
		return object.BoolOf(left.(*object.String).Value == right.(*object.String).Value), nil
	case object.KindBool:
		return object.BoolOf(left.(*object.Bool).Value == right.(*object.Bool).Value), nil
	case object.KindByte:
		return object.BoolOf(left.(*object.Byte).Value == right.(*object.Byte).Value), nil
	case object.KindChar:
		return object.BoolOf(left.(*object.Char).Value == right.(*object.Char).Value), nil
	case object.KindNil:
		return object.True, nil
	case object.KindArray:
		return object.BoolOf(object.DeepEqual(left, right)), nil
	case object.KindDict:
		return object.BoolOf(dictEqual(left.(*object.Dict), right.(*object.Dict))), nil
	case object.KindError:
		return object.BoolOf(left.(*object.Error).Message == right.(*object.Error).Message), nil
	}
	return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("不支持 %s 比较", left.Kind())}
}

func evalCompare(pos ast.Position, left, right object.Object, accept ...int) (object.Object, error) {
	if left.Kind() == object.KindArray || right.Kind() == object.KindArray {
		return nil, &EvalError{Pos: pos, Reason: "数组不支持大小比较"}
	}
	if lc, ok := left.(*object.Char); ok {
		if rc, ok := right.(*object.Char); ok {
			cmp := 0
			switch {
			case lc.Value < rc.Value:
				cmp = -1
			case lc.Value > rc.Value:
				cmp = 1
			}
			for _, a := range accept {
				if cmp == a {
					return object.True, nil
				}
			}
			return object.False, nil
		}
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("比较不支持 %s 与 %s", left.Kind(), right.Kind())}
	}
	lv := toFloat(left)
	rv := toFloat(right)
	if lv == nil || rv == nil {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("比较不支持 %s 与 %s", left.Kind(), right.Kind())}
	}
	cmp := 0
	switch {
	case *lv < *rv:
		cmp = -1
	case *lv > *rv:
		cmp = 1
	}
	for _, a := range accept {
		if cmp == a {
			return object.True, nil
		}
	}
	return object.False, nil
}

func evalAnd(pos ast.Position, left, right object.Object) (object.Object, error) {
	if left.Kind() != object.KindBool {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("且 左侧必须是布尔，实际是 %s", left.Kind())}
	}
	if !left.(*object.Bool).Value {
		return object.False, nil
	}
	if right.Kind() != object.KindBool {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("且 右侧必须是布尔，实际是 %s", right.Kind())}
	}
	return right, nil
}

func evalOr(pos ast.Position, left, right object.Object) (object.Object, error) {
	if left.Kind() != object.KindBool {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("或 左侧必须是布尔，实际是 %s", left.Kind())}
	}
	if left.(*object.Bool).Value {
		return object.True, nil
	}
	if right.Kind() != object.KindBool {
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("或 右侧必须是布尔，实际是 %s", right.Kind())}
	}
	return right, nil
}

func evalUnaryExpr(expr *ast.UnaryExpr, env *Environment) (object.Object, error) {
	right, err := Eval(expr.Right, env)
	if err != nil {
		return nil, err
	}
	pos := expr.Position
	right = unwrapNamed(right)
	switch expr.Op {
	case "-":
		switch v := right.(type) {
		case *object.Integer:
			return &object.Integer{Value: -v.Value}, nil
		case *object.Float:
			return &object.Float{Value: -v.Value}, nil
		}
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("负号不支持 %s", right.Kind())}
	case "位取反", "~":
		if n, ok := intValue(right); ok {
			return &object.Integer{Value: ^n}, nil
		}
		return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("位取反仅支持整数家族，实际是 %s", right.Kind())}
	case "非", "!":
		if right.Kind() != object.KindBool {
			return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("非 必须是布尔，实际是 %s", right.Kind())}
		}
		return object.BoolOf(!right.(*object.Bool).Value), nil
	}
	return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("未知一元运算符：%s", expr.Op)}
}

func toFloat(o object.Object) *float64 {
	o = unwrapNamed(o)
	switch v := o.(type) {
	case *object.Integer:
		f := float64(v.Value)
		return &f
	case *object.Float:
		return &v.Value
	case *object.Byte:
		f := float64(v.Value)
		return &f
	}
	return nil
}

// intValue 把整数家族（整数、字节）转为 int64。
func intValue(o object.Object) (int64, bool) {
	o = unwrapNamed(o)
	switch v := o.(type) {
	case *object.Integer:
		return v.Value, true
	case *object.Byte:
		return int64(v.Value), true
	}
	return 0, false
}

func isIntFamily(o object.Object) bool {
	o = unwrapNamed(o)
	switch o.(type) {
	case *object.Integer, *object.Byte:
		return true
	}
	return false
}

func evalReturnStmt(node *ast.ReturnStmt, env *Environment) (object.Object, error) {
	if len(node.Values) == 0 {
		return nil, &returnSignal{values: []object.Object{object.Null}}
	}
	values := make([]object.Object, 0, len(node.Values))
	for _, v := range node.Values {
		val, err := Eval(v, env)
		if err != nil {
			return nil, err
		}
		if val == nil {
			val = object.Null
		}
		values = append(values, val)
	}
	return nil, &returnSignal{values: values}
}

func evalCallExprDispatch(node *ast.CallExpr, env *Environment) (object.Object, error) {
	if ident, ok := node.Callee.(*ast.Identifier); ok && ident.Name == "捕获" && len(node.Args) == 0 {
		return evalRecover(env.Runtime()), nil
	}
	if chain, ok := collectMixfixCallChain(node); ok && env.mixfixFunctions().hasRoot(chain.root) {
		return evalMixfixCall(chain, env, node.Position)
	}
	if member, ok := node.Callee.(*ast.MemberExpr); ok {
		return evalMethodCall(node, member, env)
	}
	return evalCallExpr(node, env)
}

func evalMethodCall(node *ast.CallExpr, member *ast.MemberExpr, env *Environment) (object.Object, error) {
	if chain, ok := collectMixfixCallChain(node); ok && env.mixfixFunctions().hasRoot(chain.root) {
		return evalMixfixCall(chain, env, node.Position)
	}
	obj, err := Eval(member.Object, env)
	if err != nil {
		return nil, err
	}
	args := make([]object.Object, 0, len(node.Args))
	for _, a := range node.Args {
		v, err := Eval(a, env)
		if err != nil {
			return nil, err
		}
		if v == nil {
			v = object.Null
		}
		args = append(args, v)
	}
	args = spreadTuples(args)
	if named, ok := obj.(*object.Named); ok {
		if namedType, exists := env.Runtime().namedTypes[named.TypeName]; exists {
			if fn := namedType.lookupMethod(member.Name); fn != nil {
				return applyFunction(fn, append([]object.Object{named}, args...), node.Position)
			}
		}
	}
	if fn, ok := env.primitiveMethods().lookup(obj, member.Name); ok {
		return applyFunction(fn, append([]object.Object{obj}, args...), node.Position)
	}
	switch target := obj.(type) {
	case *Instance:
		fn, err := target.getMethod(member.Name, env)
		if err != nil {
			return nil, &EvalError{Pos: node.Position, Reason: err.Error()}
		}
		return applyFunction(fn, args, node.Position)
	case *object.Named:
		if namedType, ok := env.Runtime().namedTypes[target.TypeName]; ok {
			fn := namedType.lookupMethod(member.Name)
			if fn == nil {
				return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 没有方法 %s", target.TypeName, member.Name)}
			}
			return applyFunction(fn, append([]object.Object{target}, args...), node.Position)
		}
		return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 没有方法 %s", target.TypeName, member.Name)}
	case *object.Array:
		return evalArrayMethod(node.Position, target, member.Name, args)
	case *object.Module:
		callee, ok := target.Exports[member.Name]
		if !ok {
			return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("模块 %s 没有公开成员 %s", target.Name, member.Name)}
		}
		return callExported(env.Runtime(), callee, args, node.Position)
	default:
		return callDynamicMember(env.Runtime(), obj, member, args, node.Position)
	}
}

func callExported(rt *Runtime, callee object.Object, args []object.Object, pos ast.Position) (object.Object, error) {
	switch fn := callee.(type) {
	case *Function:
		return applyFunction(fn, args, pos)
	case *Class:
		return fn.instantiate(rt, pos, args)
	}
	return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("%s 不是可调用对象", callee.Kind())}
}

func evalArrayMethod(pos ast.Position, arr *object.Array, name string, args []object.Object) (object.Object, error) {
	switch name {
	case "长度":
		if len(args) != 0 {
			return nil, &EvalError{Pos: pos, Reason: "长度 不需要参数"}
		}
		return &object.Integer{Value: int64(len(arr.Elements))}, nil
	case "反转":
		if len(args) != 0 {
			return nil, &EvalError{Pos: pos, Reason: "反转 不需要参数"}
		}
		out := make([]object.Object, len(arr.Elements))
		for i := range arr.Elements {
			out[i] = arr.Elements[len(arr.Elements)-1-i]
		}
		return &object.Array{Elements: out}, nil
	}
	return nil, &EvalError{Pos: pos, Reason: fmt.Sprintf("数组没有方法 %s", name)}
}

func evalFieldAssign(node *ast.FieldAssignExpr, env *Environment) (object.Object, error) {
	obj, err := Eval(node.Object, env)
	if err != nil {
		return nil, err
	}
	val, err := Eval(node.Value, env)
	if err != nil {
		return nil, err
	}
	inst, ok := obj.(*Instance)
	if !ok {
		return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 不支持字段赋值", obj.Kind())}
	}
	r := inst.resolveField(node.Name)
	if !r.found {
		return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 没有字段 %s", inst.Class.Name, node.Name)}
	}
	if r.ambiguous {
		return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 的字段 %s 在嵌入链中有歧义，需通过嵌入名限定访问", inst.Class.Name, node.Name)}
	}
	vis := ownerFieldVis(r.owner, node.Name)
	if !memberAccessible(vis, r.owner, env, r.owner != inst.Class) {
		return nil, &EvalError{Pos: node.Position, Reason: fmt.Sprintf("%s 的私有字段 %s 不可在模型外访问", r.owner.Name, node.Name)}
	}
	if err := inst.setFieldValue(node.Name, val); err != nil {
		return nil, &EvalError{Pos: node.Position, Reason: err.Error()}
	}
	return val, nil
}
