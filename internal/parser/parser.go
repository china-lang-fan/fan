package parser

import (
	"fmt"
	"strconv"
	"strings"

	"fan/internal/ast"
	"fan/internal/lexer"
	"fan/internal/token"
)

type precedence int

const (
	_ precedence = iota
	LOWEST
	PREC_TERNARY
	PREC_OR
	PREC_AND
	PREC_EQUALS
	PREC_COMPARE
	PREC_SUM
	PREC_PRODUCT
	PREC_PREFIX
	PREC_POSTFIX
)

var precedences = map[token.Type]precedence{
	token.QUEST:   PREC_TERNARY,
	token.OR:      PREC_OR,
	token.AND:     PREC_AND,
	token.EQ:      PREC_EQUALS,
	token.NEQ:     PREC_EQUALS,
	token.LT:      PREC_COMPARE,
	token.LTE:     PREC_COMPARE,
	token.GT:      PREC_COMPARE,
	token.GTE:     PREC_COMPARE,
	token.MEMBER:  PREC_POSTFIX,
	token.DOT:     PREC_POSTFIX,
	token.PLUS:    PREC_SUM,
	token.MINUS:   PREC_SUM,
	token.STAR:    PREC_PRODUCT,
	token.SLASH:   PREC_PRODUCT,
	token.PERCENT: PREC_PRODUCT,
}

type parseError struct {
	Line    int
	Column  int
	Message string
}

func (e parseError) Error() string {
	return fmt.Sprintf("第%d行第%d列：%s", e.Line, e.Column, e.Message)
}

type Errors []parseError

func (es Errors) Error() string {
	if len(es) == 0 {
		return ""
	}
	return es[0].Error()
}

type Parser struct {
	l              *lexer.Lexer
	cur            token.Token
	peek           token.Token
	peek2          token.Token
	errors         Errors
	parenDepth     int
	implicitCallOK bool
}

func New(source string) *Parser {
	p := &Parser{l: lexer.New(source)}
	p.cur = p.loadNextToken()
	p.peek = p.loadNextToken()
	p.peek2 = p.loadNextToken()
	return p
}

func (p *Parser) loadNextToken() token.Token {
	return p.l.NextToken()
}

func ParseProgram(source string) (*ast.Program, Errors) {
	p := New(source)
	prog := p.ParseProgram()
	return prog, p.errors
}

func (p *Parser) Errors() Errors { return p.errors }

func (p *Parser) next() {
	p.cur = p.peek
	p.peek = p.peek2
	p.peek2 = p.loadNextToken()
	if p.parenDepth > 0 {
		for p.cur.Type == token.NEWLINE {
			p.cur = p.peek
			p.peek = p.peek2
			p.peek2 = p.loadNextToken()
		}
	}
}

func (p *Parser) addError(t token.Token, format string, args ...any) {
	p.errors = append(p.errors, parseError{
		Line:    t.Line,
		Column:  t.Column,
		Message: fmt.Sprintf(format, args...),
	})
}

func (p *Parser) ParseProgram() *ast.Program {
	prog := &ast.Program{}
	for p.cur.Type != token.EOF {
		if p.cur.Type == token.NEWLINE {
			p.next()
			continue
		}
		if p.cur.Type == token.ILLEGAL {
			p.addError(p.cur, "非法字符：%s", p.cur.Literal)
			p.next()
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			prog.Statements = append(prog.Statements, stmt)
		} else {
			for p.cur.Type != token.NEWLINE && p.cur.Type != token.EOF {
				p.next()
			}
		}
		p.skipStatementEnd()
	}
	return prog
}

func (p *Parser) skipStatementEnd() {
	for p.cur.Type == token.NEWLINE {
		p.next()
	}
}

func (p *Parser) parseStatement() ast.Statement {
	switch {
	case p.cur.Type == token.TAG:
		return p.parseTaggedStatement()
	case p.cur.Type == token.SWITCH:
		return p.parseSwitchStatement()
	case p.cur.Type == token.IF:
		return p.parseIfStatement()
	case p.cur.Type == token.WHILE:
		return p.parseWhileStatement()
	case p.cur.Type == token.REPEAT:
		return p.parseRepeatStatement()
	case p.cur.Type == token.FORIN:
		return p.parseForEachStatement()
	case p.cur.Type == token.TRY:
		return p.parseTryStatement()
	case p.cur.Type == token.IMPORT:
		return p.parseImportStatement()
	case p.cur.Type == token.EXPORT:
		return p.parseExportStatement()
	case p.cur.Type == token.DEFINE:
		p.next()
		switch {
		case p.cur.Type == token.FUNCTION:
			return p.parseFunctionStatement()
		case p.cur.Type == token.CLASS:
			return p.parseClassStatement()
		case p.cur.Type == token.IDENT && p.peek.Type == token.MEMBER:
			return p.parseIdentMemberStatement()
		default:
			return p.parseVarDecl()
		}
	case p.cur.Type == token.FUNCTION:
		return p.parseFunctionStatement()
	case p.cur.Type == token.RETURN:
		return p.parseReturnStatement()
	case p.cur.Type == token.CLASS:
		return p.parseClassStatement()
	case p.cur.Type == token.IDENT && p.peek.Type == token.MEMBER:
		return p.parseIdentMemberStatement()
	case p.cur.Type == token.BREAK:
		stmt := &ast.BreakStmt{Position: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
		p.next()
		return stmt
	case p.cur.Type == token.CONTINUE:
		stmt := &ast.ContinueStmt{Position: ast.Position{Line: p.cur.Line, Column: p.cur.Column}}
		p.next()
		return stmt
	case p.cur.Type == token.DEFINE || p.cur.Type == token.VAR || p.cur.Type == token.CONST:
		return p.parseVarDecl()
	case p.cur.Type == token.TYPE_INT || p.cur.Type == token.TYPE_FLOAT ||
		p.cur.Type == token.TYPE_STRING || p.cur.Type == token.TYPE_BOOL ||
		p.cur.Type == token.TYPE_ARRAY || p.cur.Type == token.TYPE_DICT ||
		p.cur.Type == token.TYPE_ERROR:
		if p.peek.Type == token.IDENT {
			return p.parseVarDecl()
		}
		return p.parseExpressionStatement()
	case p.cur.Type == token.IDENT && (p.peek.Type == token.ASSIGN || p.peek.Type == token.COMMA):
		return p.parseBareDecl()
	case p.cur.Type == token.IDENT && p.peek.Type == token.LBRACK:
		return p.parseIndexAssignOrExpr()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseSwitchStatement() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	var subject ast.Expression
	if p.cur.Type != token.NEWLINE {
		subject = p.parseExpression(LOWEST)
	}
	p.skipStatementEnd()
	stmt := &ast.SwitchStmt{Position: pos, Subject: subject}
	for idx := 0; ; idx++ {
		branch, ok := p.parseSwitchBranch()
		if !ok {
			return nil
		}
		if subject == nil && !branch.Guard && !branch.Default {
			p.addError(p.cur, "无匹配对象时，判断分支应使用 当")
			return nil
		}
		if branch.Default && p.cur.Type != token.END {
			p.addError(p.cur, "其他 分支必须在最后")
			return nil
		}
		stmt.Branches = append(stmt.Branches, branch)
		if p.cur.Type == token.END {
			p.next()
			return stmt
		}
		if p.cur.Type == token.EOF {
			p.addError(p.cur, "判断 语句缺少 结束")
			return nil
		}
	}
}

func (p *Parser) parseSwitchBranch() (ast.SwitchBranch, bool) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	branch := ast.SwitchBranch{Position: pos}
	switch p.cur.Type {
	case token.ASSIGN:
		p.next()
		for {
			value := p.parseExpression(LOWEST)
			if value == nil {
				return branch, false
			}
			branch.Values = append(branch.Values, value)
			if p.cur.Type != token.COMMA {
				break
			}
			p.next()
			continue
		}
	case token.WHILE:
		p.next()
		condition := p.parseExpression(LOWEST)
		if condition == nil {
			return branch, false
		}
		branch.Guard = true
		branch.Values = []ast.Expression{condition}
	case token.DEFAULT:
		p.next()
		branch.Default = true
	case token.END, token.EOF:
		p.addError(p.cur, "判断 语句缺少 结束")
		return branch, false
	default:
		p.addError(p.cur, "判断 分支应以 为、当 或 其他 开头")
		return branch, false
	}
	p.skipStatementEnd()
	body, ok := p.parseSwitchBody()
	if !ok {
		return branch, false
	}
	branch.Body = body
	return branch, true
}

func (p *Parser) parseSwitchBody() (*ast.BlockStmt, bool) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	block := &ast.BlockStmt{Position: pos}
	for {
		p.skipStatementEnd()
		switch p.cur.Type {
		case token.ASSIGN, token.WHILE, token.DEFAULT, token.END, token.EOF:
			return block, true
		case token.ILLEGAL:
			p.addError(p.cur, "非法字符：%s", p.cur.Literal)
			return nil, false
		}
		stmt := p.parseStatement()
		if stmt == nil {
			return nil, false
		}
		block.Statements = append(block.Statements, stmt)
		if p.cur.Type != token.NEWLINE &&
			p.cur.Type != token.ASSIGN && p.cur.Type != token.WHILE &&
			p.cur.Type != token.DEFAULT && p.cur.Type != token.END && p.cur.Type != token.EOF {
			p.addError(p.cur, "语句后应换行，实际是 %q", p.cur.Literal)
			return nil, false
		}
	}
}

func (p *Parser) parseIfStatement() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	stmt := &ast.IfStmt{Position: pos}

	p.next()
	branch, ok := p.parseIfBranch()
	if !ok {
		return nil
	}
	stmt.Branches = append(stmt.Branches, branch)

	for {
		switch p.cur.Type {
		case token.ELSEIF:
			p.next()
			br, ok := p.parseIfBranch()
			if !ok {
				return nil
			}
			stmt.Branches = append(stmt.Branches, br)
		case token.ELSE:
			p.next()
			if p.cur.Type == token.IF {
				p.next()
				br, ok := p.parseIfBranch()
				if !ok {
					return nil
				}
				stmt.Branches = append(stmt.Branches, br)
				continue
			}
			if p.cur.Type == token.THEN {
				p.addError(p.cur, "否则 后不能跟 那么")
				return nil
			}
			body, ok := p.parseBlock()
			if !ok {
				return nil
			}
			stmt.Else = body
			if p.cur.Type != token.END {
				p.addError(p.cur, "条件语句缺少 结束")
				return nil
			}
			p.next()
			return stmt
		case token.END:
			p.next()
			return stmt
		case token.EOF:
			p.addError(p.cur, "条件语句缺少 结束")
			return nil
		default:
			p.addError(p.cur, "条件语句中出现意外内容：%q", p.cur.Literal)
			return nil
		}
	}
}

func (p *Parser) parseIfBranch() (ast.IfBranch, bool) {
	cond := p.parseExpression(LOWEST)
	if cond == nil {
		return ast.IfBranch{}, false
	}
	if p.cur.Type != token.THEN {
		p.addError(p.cur, "如果 条件后缺少 那么")
		return ast.IfBranch{}, false
	}
	p.next()
	body, ok := p.parseBlock()
	if !ok {
		return ast.IfBranch{}, false
	}
	return ast.IfBranch{Condition: cond, Body: body}, true
}

func (p *Parser) parseBlock() (*ast.BlockStmt, bool) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	block := &ast.BlockStmt{Position: pos}
	for {
		p.skipStatementEnd()
		switch p.cur.Type {
		case token.END, token.ELSE, token.ELSEIF, token.UNTIL, token.EOF:
			return block, true
		case token.ILLEGAL:
			p.addError(p.cur, "非法字符：%s", p.cur.Literal)
			return nil, false
		}
		stmt := p.parseStatement()
		if stmt == nil {
			return nil, false
		}
		block.Statements = append(block.Statements, stmt)
		if p.cur.Type != token.NEWLINE && p.cur.Type != token.END &&
			p.cur.Type != token.ELSE && p.cur.Type != token.ELSEIF &&
			p.cur.Type != token.UNTIL && p.cur.Type != token.EOF {
			p.addError(p.cur, "语句后应换行，实际是 %q", p.cur.Literal)
			return nil, false
		}
	}
}

func (p *Parser) parseVarDecl() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	isConst := false
	declType := ast.TypeAny
	sawKeyword := false

	if p.cur.Type == token.DEFINE {
		sawKeyword = true
		p.next()
	}
	switch p.cur.Type {
	case token.VAR:
		sawKeyword = true
		p.next()
	case token.CONST:
		sawKeyword = true
		isConst = true
		p.next()
	}
	switch p.cur.Type {
	case token.TYPE_INT:
		sawKeyword = true
		declType = ast.TypeInt
		p.next()
	case token.TYPE_FLOAT:
		sawKeyword = true
		declType = ast.TypeFloat
		p.next()
	case token.TYPE_STRING:
		sawKeyword = true
		declType = ast.TypeString
		p.next()
	case token.TYPE_BOOL:
		sawKeyword = true
		declType = ast.TypeBool
		p.next()
	case token.TYPE_ARRAY:
		sawKeyword = true
		declType = ast.TypeArray
		p.next()
	case token.TYPE_DICT:
		sawKeyword = true
		declType = ast.TypeDict
		p.next()
	case token.TYPE_ERROR:
		sawKeyword = true
		declType = ast.TypeError
		p.next()
	}

	if p.cur.Type != token.IDENT {
		p.addError(p.cur, "声明语句缺少变量名")
		return nil
	}
	name := p.cur.Literal
	p.next()

	if p.cur.Type == token.COMMA {
		names := []string{name}
		for p.cur.Type == token.COMMA {
			p.next()
			if p.cur.Type != token.IDENT {
				p.addError(p.cur, "多变量声明应为 变量 名1、名2 = 值")
				return nil
			}
			names = append(names, p.cur.Literal)
			p.next()
		}
		if p.cur.Type != token.ASSIGN {
			p.addError(p.cur, "多变量声明必须使用 为 或 =")
			return nil
		}
		p.next()
		value := p.parseExpression(LOWEST)
		if value == nil {
			return nil
		}
		return &ast.MultiDecl{
			Position:   pos,
			IsConst:    isConst,
			IsExplicit: sawKeyword,
			Names:      names,
			Value:      value,
		}
	}

	if p.cur.Type != token.ASSIGN {
		p.addError(p.cur, "声明必须赋初值（使用 为 或 =）")
		return nil
	}
	p.next()

	value := p.parseExpression(LOWEST)
	if value == nil {
		return nil
	}
	return &ast.VarDecl{
		Position:   pos,
		IsConst:    isConst,
		IsExplicit: sawKeyword,
		DeclType:   declType,
		Name:       name,
		Value:      value,
	}
}

func compoundAssignOp(t token.Type) (string, bool) {
	switch t {
	case token.PLUS_EQ:
		return "加", true
	case token.MINUS_EQ:
		return "减", true
	case token.STAR_EQ:
		return "乘", true
	case token.SLASH_EQ:
		return "除", true
	}
	return "", false
}

func assignableExpr(expr ast.Expression) (ast.Expression, bool) {
	switch expr.(type) {
	case *ast.Identifier, *ast.IndexExpr, *ast.MemberExpr:
		return expr, true
	}
	return nil, false
}

func (p *Parser) parseBareDecl() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	name := p.cur.Literal
	p.next()
	if p.cur.Type == token.COMMA {
		names := []string{name}
		for p.cur.Type == token.COMMA {
			p.next()
			if p.cur.Type != token.IDENT {
				p.addError(p.cur, "多目标赋值应为 名1、名2 = 值")
				return nil
			}
			names = append(names, p.cur.Literal)
			p.next()
		}
		if p.cur.Type != token.ASSIGN {
			p.addError(p.cur, "多目标赋值应使用 为 或 =")
			return nil
		}
		p.next()
		value := p.parseExpression(LOWEST)
		if value == nil {
			return nil
		}
		return &ast.MultiAssign{Position: pos, Names: names, Value: value}
	}
	p.next()
	value := p.parseExpression(LOWEST)
	if value == nil {
		return nil
	}
	return &ast.VarDecl{
		Position:   pos,
		IsConst:    false,
		IsExplicit: false,
		DeclType:   ast.TypeAny,
		Name:       name,
		Value:      value,
	}
}

func (p *Parser) parseExpressionStatement() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.implicitCallOK = true
	expr := p.parseExpression(LOWEST)
	p.implicitCallOK = false
	if expr == nil {
		return nil
	}
	if p.cur.Type == token.ASSIGN {
		if member, ok := expr.(*ast.MemberExpr); ok {
			p.next()
			value := p.parseExpression(LOWEST)
			if value == nil {
				return nil
			}
			return &ast.FieldAssignExpr{
				Position: pos,
				Object:   member.Object,
				Name:     member.Name,
				Value:    value,
			}
		}
	}
	if op, ok := compoundAssignOp(p.cur.Type); ok {
		target, ok := assignableExpr(expr)
		if !ok {
			p.addError(p.cur, "复合赋值左侧必须是变量、下标或字段")
			return nil
		}
		p.next()
		value := p.parseExpression(LOWEST)
		if value == nil {
			return nil
		}
		return &ast.CompoundAssignStmt{Position: pos, Target: target, Op: op, Value: value}
	}
	return &ast.ExpressionStmt{Position: pos, Expression: expr}
}

func (p *Parser) parseExpression(prec precedence) ast.Expression {
	left := p.parsePrefix()
	if left == nil {
		return nil
	}
	left = p.parsePostfix(left)
	if left == nil {
		return nil
	}
	for {
		if p.cur.Type == token.NEWLINE || p.cur.Type == token.EOF || p.cur.Type == token.COMMA ||
			p.cur.Type == token.RPAREN || p.cur.Type == token.RBRACK || p.cur.Type == token.END ||
			p.cur.Type == token.THEN || p.cur.Type == token.ELSE || p.cur.Type == token.ELSEIF ||
			p.cur.Type == token.LOOP || p.cur.Type == token.TIMES || p.cur.Type == token.UNTIL ||
			p.cur.Type == token.INOF || p.cur.Type == token.COLON {
			if p.implicitCallOK && isImplicitCallEnd(p.cur.Type) {
				allowIdentifier := p.cur.Type == token.NEWLINE || p.cur.Type == token.EOF
				switch expr := left.(type) {
				case *ast.Identifier:
					if allowIdentifier {
						return &ast.MaybeCallExpr{Position: left.Pos(), Callee: expr, AutoCall: true}
					}
				case *ast.MemberExpr:
					return &ast.MaybeCallExpr{Position: left.Pos(), Callee: expr, AutoCall: true}
				}
			}
			return left
		}
		if p.cur.Type == token.MEMBER || p.cur.Type == token.DOT {
			left = p.parseMember(left)
			continue
		}
		if p.implicitCallOK && p.cur.Type == token.IDENT {
			if p.currentPrecedence() == LOWEST {
				left = p.parseImplicitCall(left)
				continue
			}
		}
		if p.implicitCallOK && isImplicitCallContinuation(p.cur.Type) && p.currentPrecedence() == LOWEST {
			left = p.parseImplicitCall(left)
			continue
		}
		if prec >= p.currentPrecedence() {
			return left
		}
		left = p.parseInfix(left)
		if left == nil {
			return nil
		}
	}
}

func isImplicitCallEnd(t token.Type) bool {
	switch t {
	case token.NEWLINE, token.EOF, token.COMMA, token.RPAREN, token.RBRACK, token.THEN, token.ELSE, token.ELSEIF:
		return true
	}
	return false
}

func (p *Parser) parseImplicitCall(head ast.Expression) ast.Expression {
	if p.cur.Type == token.IDENT {
		if _, isIdent := head.(*ast.Identifier); isIdent && p.peek.Type != token.MEMBER {
			ident, _ := head.(*ast.Identifier)
			method := p.cur.Literal
			pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
			p.next()
			args, ok := p.parseImplicitArgs()
			if !ok {
				return nil
			}
			fullArgs := append([]ast.Expression{ident}, args...)
			return &ast.CallExpr{
				Position: pos,
				Callee:   &ast.Identifier{Position: pos, Name: method},
				Args:     fullArgs,
			}
		}
	}
	args, ok := p.parseImplicitArgs()
	if !ok {
		return nil
	}
	if len(args) == 0 {
		return head
	}
	pos := head.Pos()
	callee, ok := head.(*ast.Identifier)
	if !ok {
		p.addError(p.cur, "无括号调用必须以函数名开头")
		return nil
	}
	return &ast.CallExpr{Position: pos, Callee: callee, Args: args}
}

func (p *Parser) parseImplicitArgs() ([]ast.Expression, bool) {
	var args []ast.Expression
	p.implicitCallOK = false
	for {
		if p.cur.Type == token.COMMA {
			p.next()
			continue
		}
		if p.cur.Type == token.NEWLINE || p.cur.Type == token.EOF ||
			p.cur.Type == token.RPAREN || p.cur.Type == token.RBRACK || p.cur.Type == token.END ||
			p.cur.Type == token.THEN || p.cur.Type == token.ELSE ||
			p.cur.Type == token.ELSEIF || p.cur.Type == token.INOF ||
			p.cur.Type == token.LOOP || p.cur.Type == token.TIMES ||
			p.cur.Type == token.UNTIL {
			p.implicitCallOK = true
			return args, true
		}
		arg := p.parseExpression(LOWEST)
		if arg == nil {
			p.implicitCallOK = true
			return nil, false
		}
		args = append(args, arg)
	}
}

func (p *Parser) parsePostfix(left ast.Expression) ast.Expression {
	for {
		switch p.cur.Type {
		case token.LBRACK:
			pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
			p.parenDepth++
			p.next()
			idx := p.parseExpression(LOWEST)
			if idx == nil {
				p.parenDepth--
				return nil
			}
			if p.cur.Type != token.RBRACK {
				p.addError(p.cur, "缺少右方括号 ]")
				p.parenDepth--
				return nil
			}
			p.parenDepth--
			p.next()
			left = &ast.IndexExpr{Position: pos, Target: left, Index: idx}
		case token.LPAREN:
			pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
			args, ok := p.parseArgs()
			if !ok {
				return nil
			}
			left = &ast.CallExpr{Position: pos, Callee: left, Args: args}
		case token.INC:
			pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
			left = &ast.UpdateExpr{Position: pos, Target: left, Op: "加", Prefix: false}
			p.next()
		case token.DEC:
			pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
			left = &ast.UpdateExpr{Position: pos, Target: left, Op: "减", Prefix: false}
			p.next()
		default:
			return left
		}
	}
}

func (p *Parser) parseArgs() ([]ast.Expression, bool) {
	p.parenDepth++
	p.next()
	var args []ast.Expression
	for {
		if p.cur.Type == token.RPAREN {
			p.parenDepth--
			p.next()
			return args, true
		}
		arg := p.parseExpression(LOWEST)
		if arg == nil {
			p.parenDepth--
			return nil, false
		}
		args = append(args, arg)
		if p.cur.Type == token.COMMA {
			p.next()
			continue
		}
		if p.cur.Type == token.RPAREN {
			p.parenDepth--
			p.next()
			return args, true
		}
		p.addError(p.cur, "参数列表应以 , 分隔或以 ) 结束")
		p.parenDepth--
		return nil, false
	}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.parenDepth++
	p.next()
	var elems []ast.Expression
	for {
		if p.cur.Type == token.RBRACK {
			p.parenDepth--
			p.next()
			return &ast.ArrayLiteral{Position: pos, Elements: elems}
		}
		el := p.parseExpression(LOWEST)
		if el == nil {
			p.parenDepth--
			return nil
		}
		elems = append(elems, el)
		if p.cur.Type == token.COMMA {
			p.next()
			continue
		}
		if p.cur.Type == token.RBRACK {
			p.parenDepth--
			p.next()
			return &ast.ArrayLiteral{Position: pos, Elements: elems}
		}
		p.addError(p.cur, "数组元素应以 , 分隔或以 ] 结束")
		p.parenDepth--
		return nil
	}
}

func (p *Parser) parseDictLiteral() ast.Expression {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.parenDepth++
	p.next()
	var pairs []ast.DictPair
	for {
		if p.cur.Type == token.RBRACE {
			p.parenDepth--
			p.next()
			return &ast.DictLiteral{Position: pos, Pairs: pairs}
		}
		key := p.parseExpression(LOWEST)
		if key == nil {
			p.parenDepth--
			return nil
		}
		if p.cur.Type != token.COLON {
			p.addError(p.cur, "字典项应以 : 分隔键和值")
			p.parenDepth--
			return nil
		}
		p.next()
		val := p.parseExpression(LOWEST)
		if val == nil {
			p.parenDepth--
			return nil
		}
		pairs = append(pairs, ast.DictPair{Key: key, Value: val})
		if p.cur.Type == token.COMMA {
			p.next()
			continue
		}
		if p.cur.Type == token.RBRACE {
			p.parenDepth--
			p.next()
			return &ast.DictLiteral{Position: pos, Pairs: pairs}
		}
		p.addError(p.cur, "字典项应以 , 分隔或以 } 结束")
		p.parenDepth--
		return nil
	}
}

func (p *Parser) currentPrecedence() precedence {
	if pr, ok := precedences[p.cur.Type]; ok {
		return pr
	}
	return LOWEST
}

func (p *Parser) parsePrefix() ast.Expression {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	switch p.cur.Type {
	case token.INT:
		v, err := strconv.ParseInt(p.cur.Literal, 10, 64)
		if err != nil {
			p.addError(p.cur, "整数超出范围：%s", p.cur.Literal)
			return nil
		}
		node := &ast.IntegerLiteral{Position: pos, Value: v}
		p.next()
		return node
	case token.FLOAT:
		v, err := strconv.ParseFloat(p.cur.Literal, 64)
		if err != nil {
			p.addError(p.cur, "小数格式错误：%s", p.cur.Literal)
			return nil
		}
		node := &ast.FloatLiteral{Position: pos, Value: v}
		p.next()
		return node
	case token.STRING:
		node := &ast.StringLiteral{Position: pos, Value: p.cur.Literal}
		p.next()
		return node
	case token.TRUE:
		node := &ast.BoolLiteral{Position: pos, Value: true}
		p.next()
		return node
	case token.FALSE:
		node := &ast.BoolLiteral{Position: pos, Value: false}
		p.next()
		return node
	case token.NIL:
		node := &ast.NilLiteral{Position: pos}
		p.next()
		return node
	case token.IDENT, token.SELF:
		node := &ast.Identifier{Position: pos, Name: p.cur.Literal}
		p.next()
		return node
	case token.TYPE_ERROR:
		node := &ast.Identifier{Position: pos, Name: "错误"}
		p.next()
		return node
	case token.CHECK:
		return p.parseCheckExpression()
	case token.INC:
		p.next()
		implicit := p.implicitCallOK
		p.implicitCallOK = false
		target := p.parseExpression(PREC_POSTFIX)
		p.implicitCallOK = implicit
		if target == nil {
			return nil
		}
		return &ast.UpdateExpr{Position: pos, Target: target, Op: "加", Prefix: true}
	case token.DEC:
		p.next()
		implicit := p.implicitCallOK
		p.implicitCallOK = false
		target := p.parseExpression(PREC_POSTFIX)
		p.implicitCallOK = implicit
		if target == nil {
			return nil
		}
		return &ast.UpdateExpr{Position: pos, Target: target, Op: "减", Prefix: true}
	case token.MINUS:
		op := "-"
		p.next()
		right := p.parseExpression(PREC_PREFIX)
		if right == nil {
			return nil
		}
		return &ast.UnaryExpr{Position: pos, Op: op, Right: right}
	case token.NOT:
		op := p.cur.Literal
		p.next()
		right := p.parseExpression(PREC_PREFIX)
		if right == nil {
			return nil
		}
		return &ast.UnaryExpr{Position: pos, Op: op, Right: right}
	case token.LBRACK:
		return p.parseArrayLiteral()
	case token.LBRACE:
		return p.parseDictLiteral()
	case token.FUNCTION:
		return p.parseFunctionLiteral()
	case token.LPAREN:
		p.parenDepth++
		p.next()
		expr := p.parseExpression(LOWEST)
		if expr == nil {
			p.parenDepth--
			return nil
		}
		if p.cur.Type != token.RPAREN {
			p.addError(p.cur, "缺少右括号 )")
			p.parenDepth--
			return nil
		}
		p.parenDepth--
		p.next()
		return expr
	default:
		p.addError(p.cur, "此处不期望出现：%q", p.cur.Literal)
		return nil
	}
}

func (p *Parser) parseInfix(left ast.Expression) ast.Expression {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	if p.cur.Type == token.QUEST {
		p.next()
		thenExpr := p.parseExpression(LOWEST)
		if thenExpr == nil {
			return nil
		}
		if p.cur.Type != token.COLON {
			p.addError(p.cur, "三元表达式缺少 :")
			return nil
		}
		p.next()
		elseExpr := p.parseExpression(LOWEST)
		if elseExpr == nil {
			return nil
		}
		return &ast.TernaryExpr{Position: pos, Cond: left, Then: thenExpr, Else: elseExpr}
	}
	op := p.cur.Literal
	pr := p.currentPrecedence()
	p.next()
	right := p.parseExpression(pr)
	if right == nil {
		return nil
	}
	return &ast.BinaryExpr{Position: pos, Op: op, Left: left, Right: right}
}

func (p *Parser) parseIndexAssignOrExpr() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil
	}
	if p.cur.Type != token.ASSIGN {
		if op, ok := compoundAssignOp(p.cur.Type); ok {
			p.next()
			value := p.parseExpression(LOWEST)
			if value == nil {
				return nil
			}
			return &ast.CompoundAssignStmt{Position: pos, Target: expr, Op: op, Value: value}
		}
		return &ast.ExpressionStmt{Position: pos, Expression: expr}
	}
	idx, ok := expr.(*ast.IndexExpr)
	if !ok {
		p.addError(p.cur, "赋值左侧必须是变量或数组元素")
		return nil
	}
	p.next()
	value := p.parseExpression(LOWEST)
	if value == nil {
		return nil
	}
	return &ast.IndexAssignStmt{
		Position: pos,
		Target:   idx.Target,
		Index:    idx.Index,
		Value:    value,
	}
}

func (p *Parser) parseWhileStatement() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	cond := p.parseExpression(LOWEST)
	if cond == nil {
		return nil
	}
	if p.cur.Type != token.LOOP {
		p.addError(p.cur, "当 条件后缺少 循环")
		return nil
	}
	p.next()
	body, ok := p.parseBlock()
	if !ok {
		return nil
	}
	if p.cur.Type != token.END {
		p.addError(p.cur, "循环缺少 结束")
		return nil
	}
	p.next()
	return &ast.WhileStmt{Position: pos, Condition: cond, Body: body}
}

func (p *Parser) parseRepeatStatement() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	if p.peek.Type != token.NEWLINE {
		return p.parseTimesStatement(pos)
	}
	p.next()
	body, ok := p.parseBlock()
	if !ok {
		return nil
	}
	if p.cur.Type != token.UNTIL {
		p.addError(p.cur, "重复 循环缺少 直到 条件")
		return nil
	}
	p.next()
	cond := p.parseExpression(LOWEST)
	if cond == nil {
		return nil
	}
	p.skipStatementEnd()
	if p.cur.Type != token.END {
		p.addError(p.cur, "重复 循环缺少 结束")
		return nil
	}
	p.next()
	return &ast.RepeatStmt{Position: pos, Body: body, Until: cond}
}

func (p *Parser) parseTimesStatement(pos ast.Position) ast.Statement {
	p.next()
	count := p.parseExpression(LOWEST)
	if count == nil {
		return nil
	}
	if p.cur.Type != token.TIMES {
		p.addError(p.cur, "重复 次数后缺少 次")
		return nil
	}
	p.next()
	body, ok := p.parseBlock()
	if !ok {
		return nil
	}
	if p.cur.Type == token.UNTIL {
		p.addError(p.cur, "重复 N 次 循环中不能使用 直到")
		return nil
	}
	if p.cur.Type != token.END {
		p.addError(p.cur, "重复 N 次 循环缺少 结束")
		return nil
	}
	p.next()
	return &ast.TimesStmt{Position: pos, Count: count, Body: body}
}

func (p *Parser) parseForEachStatement() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	iterable := p.parseExpression(LOWEST)
	if iterable == nil {
		return nil
	}
	if p.cur.Type != token.INOF {
		p.addError(p.cur, "遍历 语句缺少 中的")
		return nil
	}
	p.next()
	if p.cur.Type != token.IDENT {
		p.addError(p.cur, "遍历 需要元素变量名")
		return nil
	}
	firstName := p.cur.Literal
	p.next()
	secondName := ""
	if p.cur.Type == token.IDENT {
		secondName = p.cur.Literal
		p.next()
	}
	body, ok := p.parseBlock()
	if !ok {
		return nil
	}
	if p.cur.Type != token.END {
		p.addError(p.cur, "遍历 语句缺少 结束")
		return nil
	}
	p.next()
	return &ast.ForEachStmt{
		Position:   pos,
		Iterable:   iterable,
		FirstName:  firstName,
		SecondName: secondName,
		Body:       body,
	}
}

func (p *Parser) parseImportStatement() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	if p.cur.Type != token.STRING {
		p.addError(p.cur, "导入 后应为字符串路径")
		return nil
	}
	path := p.cur.Literal
	p.next()
	name := ""
	if p.cur.Type == token.IDENT && p.cur.Literal == "作为" {
		p.next()
		if p.cur.Type != token.IDENT {
			p.addError(p.cur, "作为 后应为模块名")
			return nil
		}
		name = p.cur.Literal
		p.next()
	}
	if name == "" {
		name = moduleBaseName(path)
	}
	return &ast.ImportStmt{Position: pos, Path: path, Name: name}
}

func moduleBaseName(path string) string {
	base := path
	if i := strings.LastIndexAny(path, "/\\"); i >= 0 {
		base = path[i+1:]
	}
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	return base
}

func (p *Parser) parseExportStatement() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	var inner ast.Statement
	switch p.cur.Type {
	case token.FUNCTION:
		inner = p.parseFunctionStatement()
	case token.CLASS:
		inner = p.parseClassStatement()
	case token.IDENT:
		if p.peek.Type == token.MEMBER {
			inner = p.parseMethodDefinition()
		}
	case token.DEFINE:
		p.next()
		switch {
		case p.cur.Type == token.FUNCTION:
			inner = p.parseFunctionStatement()
		case p.cur.Type == token.CLASS:
			inner = p.parseClassStatement()
		case p.cur.Type == token.IDENT && p.peek.Type == token.MEMBER:
			inner = p.parseMethodDefinition()
		default:
			inner = p.parseVarDecl()
		}
	case token.VAR, token.CONST:
		inner = p.parseVarDecl()
	default:
		p.addError(p.cur, "导出 后应为 变量/常量/函数/模型")
		return nil
	}
	if inner == nil {
		return nil
	}
	name := exportedName(inner)
	if name == "" {
		p.addError(p.cur, "导出 语句必须带名字")
		return nil
	}
	return &ast.ExportStmt{Position: pos, Name: name, Inner: inner}
}

func exportedName(stmt ast.Statement) string {
	switch n := stmt.(type) {
	case *ast.VarDecl:
		return n.Name
	case *ast.ClassStmt:
		return n.Name
	case *ast.MethodDef:
		return n.MethodName
	}
	return ""
}

func (p *Parser) parseCheckExpression() ast.Expression {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	checkTok := p.cur
	p.next()
	call := p.parseExpression(LOWEST)
	if call == nil {
		p.addError(checkTok, "检查 后应为函数调用表达式")
		return nil
	}
	return &ast.CheckExpr{Position: pos, Call: call}
}

func (p *Parser) parseTryStatement() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	if p.cur.Type != token.TRY {
		p.addError(p.cur, "尝试 语句应以 尝试 开头")
		return nil
	}
	if p.peek.Type != token.NEWLINE && p.peek.Type != token.EOF {
		p.addError(p.peek, "尝试 后应换行")
		return nil
	}
	p.next()
	p.skipStatementEnd()
	body := &ast.BlockStmt{Position: pos}
	for {
		if p.cur.Type == token.CATCH {
			break
		}
		if p.cur.Type == token.END || p.cur.Type == token.EOF {
			p.addError(p.cur, "尝试 语句缺少 捕获")
			return nil
		}
		stmt := p.parseStatement()
		if stmt == nil {
			return nil
		}
		body.Statements = append(body.Statements, stmt)
		p.skipStatementEnd()
	}

	p.next()
	catchName := "错误"
	if p.cur.Type == token.IDENT {
		catchName = p.cur.Literal
		p.next()
	}
	if p.cur.Type != token.NEWLINE && p.cur.Type != token.EOF {
		p.addError(p.cur, "捕获 后应换行")
		return nil
	}
	p.skipStatementEnd()
	catch := &ast.BlockStmt{Position: pos}
	for {
		if p.cur.Type == token.END {
			p.next()
			return &ast.TryStmt{Position: pos, Body: body, CatchName: catchName, Catch: catch}
		}
		if p.cur.Type == token.EOF {
			p.addError(p.cur, "尝试 语句缺少 结束")
			return nil
		}
		stmt := p.parseStatement()
		if stmt == nil {
			return nil
		}
		catch.Statements = append(catch.Statements, stmt)
		p.skipStatementEnd()
	}
}

func (p *Parser) parseMember(left ast.Expression) ast.Expression {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	if p.cur.Type != token.IDENT {
		p.addError(p.cur, "的 后面需要成员名")
		return nil
	}
	name := p.cur.Literal
	node := &ast.MemberExpr{Position: pos, Object: left, Name: name}
	p.next()
	if p.cur.Type == token.LPAREN {
		args, ok := p.parseArgs()
		if !ok {
			return nil
		}
		return p.parsePostfix(&ast.CallExpr{Position: pos, Callee: node, Args: args})
	}
	return p.parsePostfix(node)
}

func (p *Parser) parseTaggedStatement() ast.Statement {
	var tags []*ast.TagExpr
	for p.cur.Type == token.TAG {
		tag := p.parseTagExpr()
		if tag != nil {
			tags = append(tags, tag)
		}
		for p.cur.Type == token.NEWLINE {
			p.next()
		}
	}
	stmt := p.parseStatement()
	switch target := stmt.(type) {
	case *ast.VarDecl:
		literal, ok := target.Value.(*ast.FunctionLiteral)
		if !ok {
			pos := astPos(tags)
			p.addError(token.Token{Line: pos.Line, Column: pos.Column}, "标签只能用于模型、函数或方法")
			break
		}
		literal.Tags = append(literal.Tags, tags...)
	case *ast.MethodDef:
		target.Function.Tags = append(target.Function.Tags, tags...)
	case *ast.ExportStmt:
		if decl, ok := target.Inner.(*ast.VarDecl); ok {
			if literal, ok := decl.Value.(*ast.FunctionLiteral); ok {
				literal.Tags = append(literal.Tags, tags...)
			}
		}
	case *ast.ClassStmt:
		target.Tags = append(target.Tags, tags...)
	default:
		pos := astPos(tags)
		p.addError(token.Token{Line: pos.Line, Column: pos.Column}, "标签只能用于模型、函数或方法")
	}
	return stmt
}

func astPos(tags []*ast.TagExpr) ast.Position {
	if len(tags) > 0 {
		return tags[0].Position
	}
	return ast.Position{}
}

func (p *Parser) parseTagExpr() *ast.TagExpr {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	if p.cur.Type != token.IDENT {
		p.addError(p.cur, "标签后应为标签名称")
		return nil
	}
	tag := &ast.TagExpr{Position: pos, TypeName: p.cur.Literal}
	p.next()
	if p.cur.Type != token.LPAREN {
		return tag
	}
	p.parenDepth++
	p.next()
	for p.cur.Type != token.RPAREN && p.cur.Type != token.EOF {
		if (p.cur.Type == token.IDENT || p.cur.Type == token.METHOD) && p.peek.Type == token.ASSIGN {
			name := p.cur.Literal
			p.next()
			p.next()
			value := p.parseExpression(LOWEST)
			tag.Named = append(tag.Named, ast.NamedTagArg{Name: name, Value: value})
		} else {
			value := p.parseExpression(LOWEST)
			tag.Positional = append(tag.Positional, value)
		}
		if p.cur.Type == token.COMMA {
			p.next()
			continue
		}
		break
	}
	if p.cur.Type != token.RPAREN {
		p.addError(p.cur, "标签参数缺少右括号")
		p.parenDepth--
		return tag
	}
	p.parenDepth--
	p.next()
	return tag
}

func (p *Parser) parseFunctionStatement() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	if p.cur.Type != token.IDENT {
		p.addError(p.cur, "函数声明需要名字")
		return nil
	}
	name := p.cur.Literal
	p.next()
	params, ok := p.parseParamList()
	if !ok {
		return nil
	}
	retTypes, ok := p.parseReturnTypes()
	if !ok {
		return nil
	}
	body, ok := p.parseFunctionBody()
	if !ok {
		return nil
	}
	return &ast.VarDecl{
		Position:   pos,
		IsExplicit: false,
		Name:       name,
		Value: &ast.FunctionLiteral{
			Position:    pos,
			Params:      params,
			Body:        body,
			Name:        name,
			ReturnTypes: retTypes,
			Tags:        nil,
		},
	}
}

func (p *Parser) parseFunctionLiteral() *ast.FunctionLiteral {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	params, ok := p.parseParamList()
	if !ok {
		return nil
	}
	retTypes, ok := p.parseReturnTypes()
	if !ok {
		return nil
	}
	body, ok := p.parseFunctionBody()
	if !ok {
		return nil
	}
	return &ast.FunctionLiteral{Position: pos, Params: params, Body: body, ReturnTypes: retTypes}
}

func (p *Parser) parseReturnTypes() ([]ast.DeclType, bool) {
	if p.cur.Type != token.ARROW {
		return nil, true
	}
	p.next()
	if p.cur.Type == token.LPAREN {
		p.parenDepth++
		p.next()
		var types []ast.DeclType
		for {
			if p.cur.Type == token.RPAREN {
				p.parenDepth--
				p.next()
				return types, true
			}
			if !isTypeToken(p.cur.Type) && p.cur.Type != token.CLASS && p.cur.Type != token.IDENT {
				p.addError(p.cur, "括号内应为返回类型")
				p.parenDepth--
				return nil, false
			}
			types = append(types, ast.DeclType(p.cur.Literal))
			p.next()
			if p.cur.Type == token.COMMA {
				p.next()
				continue
			}
			if p.cur.Type == token.RPAREN {
				p.parenDepth--
				p.next()
				return types, true
			}
			p.addError(p.cur, "返回类型应以 , 分隔或以 ) 结束")
			p.parenDepth--
			return nil, false
		}
	}
	if !isTypeToken(p.cur.Type) && p.cur.Type != token.CLASS && p.cur.Type != token.IDENT {
		p.addError(p.cur, "-> 后应为返回类型")
		return nil, false
	}
	types := []ast.DeclType{ast.DeclType(p.cur.Literal)}
	p.next()
	return types, true
}

func (p *Parser) parseExpressionList() ([]ast.Expression, bool) {
	var exprs []ast.Expression
	for {
		expr := p.parseExpression(LOWEST)
		if expr == nil {
			return nil, false
		}
		exprs = append(exprs, expr)
		if p.cur.Type == token.COMMA {
			p.next()
			continue
		}
		return exprs, true
	}
}

func (p *Parser) parseParamList() ([]ast.Parameter, bool) {
	var params []ast.Parameter
	if p.cur.Type != token.LPAREN {
		return nil, false
	}
	p.parenDepth++
	p.next()
	sawDefault := false
	for {
		if p.cur.Type == token.RPAREN {
			p.parenDepth--
			p.next()
			return params, true
		}
		if p.cur.Type != token.IDENT {
			p.addError(p.cur, "参数名应为标识符")
			p.parenDepth--
			return nil, false
		}
		name := p.cur.Literal
		p.next()
		variadic := false
		if p.cur.Type == token.DOT && p.peek.Type == token.DOT && p.peek2.Type == token.DOT {
			variadic = true
			p.next()
			p.next()
			p.next()
		}
		dt := ast.TypeAny
		if variadic {
			if isTypeToken(p.cur.Type) {
				dt = parseDeclType(p.cur.Type)
				p.next()
			} else if p.cur.Type == token.CLASS {
				dt = ast.DeclType(p.cur.Literal)
				p.next()
			}
		} else if isTypeToken(p.cur.Type) {
			dt = parseDeclType(p.cur.Type)
			p.next()
		} else if p.cur.Type == token.CLASS || p.cur.Type == token.IDENT {
			dt = ast.DeclType(p.cur.Literal)
			p.next()
		}
		var def ast.Expression
		if !variadic && p.cur.Type == token.ASSIGN {
			sawDefault = true
			p.next()
			def = p.parseExpression(LOWEST)
			if def == nil {
				p.parenDepth--
				return nil, false
			}
		} else if sawDefault && !variadic {
			p.addError(p.cur, "默认参数后面不能再出现普通参数")
			p.parenDepth--
			return nil, false
		}
		params = append(params, ast.Parameter{Name: name, Type: dt, Variadic: variadic, Default: def})
		if variadic {
			if p.cur.Type == token.COMMA {
				p.addError(p.cur, "可变参数必须是最后一个参数")
				p.parenDepth--
				return nil, false
			}
			if p.cur.Type != token.RPAREN {
				p.addError(p.cur, "可变参数必须是最后一个参数")
				p.parenDepth--
				return nil, false
			}
			p.parenDepth--
			p.next()
			return params, true
		}
		if p.cur.Type == token.COMMA {
			p.next()
			continue
		}
		if p.cur.Type == token.RPAREN {
			p.parenDepth--
			p.next()
			return params, true
		}
		p.addError(p.cur, "参数列表应以 , 分隔或以 ) 结束")
		p.parenDepth--
		return nil, false
	}
}

func (p *Parser) parseFunctionBody() (*ast.BlockStmt, bool) {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	if p.cur.Type != token.NEWLINE && p.cur.Type != token.EOF {
		p.addError(p.cur, "函数体应另起一行")
		return nil, false
	}
	p.skipStatementEnd()
	block := &ast.BlockStmt{Position: pos}
	for {
		switch p.cur.Type {
		case token.END:
			p.next()
			return block, true
		case token.EOF:
			p.addError(p.cur, "函数缺少 结束")
			return nil, false
		case token.ILLEGAL:
			p.addError(p.cur, "非法字符：%s", p.cur.Literal)
			return nil, false
		}
		stmt := p.parseStatement()
		if stmt == nil {
			return nil, false
		}
		block.Statements = append(block.Statements, stmt)
		if p.cur.Type != token.NEWLINE && p.cur.Type != token.END && p.cur.Type != token.EOF {
			p.addError(p.cur, "语句后应换行，实际是 %q", p.cur.Literal)
			return nil, false
		}
		p.skipStatementEnd()
	}
}

func (p *Parser) parseReturnStatement() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	if p.cur.Type == token.NEWLINE || p.cur.Type == token.EOF || p.cur.Type == token.END {
		return &ast.ReturnStmt{Position: pos}
	}
	values, ok := p.parseExpressionList()
	if !ok {
		return nil
	}
	return &ast.ReturnStmt{Position: pos, Values: values}
}

func isImplicitCallContinuation(t token.Type) bool {
	switch t {
	case token.INT, token.FLOAT, token.STRING, token.TRUE, token.FALSE, token.NIL,
		token.LPAREN, token.LBRACK, token.MINUS, token.NOT:
		return true
	}
	return false
}

func (p *Parser) parseClassStatement() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	p.next()
	if p.cur.Type != token.IDENT {
		p.addError(p.cur, "模型 需要名字")
		return nil
	}
	name := p.cur.Literal
	p.next()
	cls := &ast.ClassStmt{
		Position: pos,
		Name:     name,
	}
	if p.cur.Type != token.NEWLINE && p.cur.Type != token.EOF {
		p.addError(p.cur, "模型名后应换行")
		return nil
	}
	p.skipStatementEnd()
	for {
		switch p.cur.Type {
		case token.END:
			p.next()
			return cls
		case token.EOF:
			p.addError(p.cur, "模型缺少 结束")
			return nil
		case token.ILLEGAL:
			p.addError(p.cur, "非法字符：%s", p.cur.Literal)
			return nil
		case token.VAR:
			p.next()
			dt := ast.TypeAny
			if isTypeToken(p.cur.Type) {
				dt = parseDeclType(p.cur.Type)
				p.next()
			}
			if p.cur.Type != token.IDENT {
				p.addError(p.cur, "字段名应为标识符")
				return nil
			}
			cls.Fields = append(cls.Fields, ast.FieldDecl{Name: p.cur.Literal, Type: dt})
			p.next()
			if p.cur.Type != token.NEWLINE && p.cur.Type != token.EOF {
				p.addError(p.cur, "字段声明应换行结束")
				return nil
			}
			p.skipStatementEnd()
		case token.EMBED:
			p.next()
			if p.cur.Type != token.IDENT {
				p.addError(p.cur, "嵌入需要类型名")
				return nil
			}
			cls.Embeds = append(cls.Embeds, p.cur.Literal)
			p.next()
			if p.cur.Type != token.NEWLINE && p.cur.Type != token.EOF {
				p.addError(p.cur, "嵌入声明应换行结束")
				return nil
			}
			p.skipStatementEnd()
		default:
			p.addError(p.cur, "模型内只能包含 字段/嵌入，实际：%q", p.cur.Literal)
			return nil
		}
	}
}

func (p *Parser) parseIdentMemberStatement() ast.Statement {
	if p.peekTwo().Type == token.METHOD {
		return p.parseMethodDefinition()
	}
	return p.parseExpressionStatement()
}

func (p *Parser) peekTwo() token.Token {
	return p.peek2
}

func (p *Parser) parseMethodDefinition() ast.Statement {
	pos := ast.Position{Line: p.cur.Line, Column: p.cur.Column}
	className := p.cur.Literal
	p.next()
	if p.cur.Type != token.MEMBER {
		p.addError(p.cur, "方法定义应为：定义 模型名 的 方法 …")
		return nil
	}
	p.next()
	if p.cur.Type != token.METHOD {
		p.addError(p.cur, "模型成员定义应使用 方法 关键字")
		return nil
	}
	p.next()
	if p.cur.Type != token.IDENT {
		p.addError(p.cur, "方法需要名字")
		return nil
	}
	methodName := p.cur.Literal
	p.next()
	params, ok := p.parseParamList()
	if !ok {
		return nil
	}
	retTypes, ok := p.parseReturnTypes()
	if !ok {
		return nil
	}
	body, ok := p.parseFunctionBody()
	if !ok {
		return nil
	}
	return &ast.MethodDef{
		Position:   pos,
		ClassName:  className,
		MethodName: methodName,
		Function: &ast.FunctionLiteral{
			Position:    pos,
			Params:      params,
			Body:        body,
			Name:        methodName,
			ReturnTypes: retTypes,
		},
	}
}

func parseDeclType(t token.Type) ast.DeclType {
	switch t {
	case token.TYPE_INT:
		return ast.TypeInt
	case token.TYPE_FLOAT:
		return ast.TypeFloat
	case token.TYPE_STRING:
		return ast.TypeString
	case token.TYPE_BOOL:
		return ast.TypeBool
	case token.TYPE_ARRAY:
		return ast.TypeArray
	case token.TYPE_DICT:
		return ast.TypeDict
	case token.TYPE_ERROR:
		return ast.TypeError
	}
	return ast.TypeAny
}

func isTypeToken(t token.Type) bool {
	return t == token.TYPE_INT || t == token.TYPE_FLOAT ||
		t == token.TYPE_STRING || t == token.TYPE_BOOL ||
		t == token.TYPE_ARRAY || t == token.TYPE_DICT || t == token.TYPE_ERROR
}
