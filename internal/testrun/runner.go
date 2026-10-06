package testrun

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fan/internal/ast"
	"fan/internal/evaluator"
	"fan/internal/parser"
)

type TestCase struct {
	Name string
	Pos  ast.Position
	Fn   *evaluator.Function
}

type Failure struct {
	Name   string
	Pos    ast.Position
	Reason string
}

type FileResult struct {
	Path       string
	Tests      []TestCase
	Failures   []Failure
	SetupError string
}

func (r FileResult) Failed() bool {
	return r.SetupError != "" || len(r.Failures) > 0
}

type Runner struct {
	Out io.Writer
}

func (r Runner) Run(target string) ([]FileResult, error) {
	root, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}
	paths, err := discoverFiles(root)
	if err != nil {
		return nil, err
	}
	results := make([]FileResult, 0, len(paths))
	for _, path := range paths {
		results = append(results, r.runFile(path, root))
	}
	r.print(results)
	return results, nil
}

func discoverFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if isTestFile(root) {
			return []string{root}, nil
		}
		return nil, nil
	}
	paths := []string{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if isTestFile(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func isTestFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, "_测试.凡") || strings.HasSuffix(base, "_test.凡") || strings.HasSuffix(base, "_测试.fan") || strings.HasSuffix(base, "_test.fan")
}

func (r Runner) runFile(path string, root string) FileResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return FileResult{Path: path, SetupError: err.Error()}
	}
	prog, errs := parser.ParseProgram(string(data))
	if len(errs) > 0 {
		messages := make([]string, 0, len(errs))
		for _, e := range errs {
			messages = append(messages, e.Error())
		}
		return FileResult{Path: path, SetupError: strings.Join(messages, "\n")}
	}
	env := evaluator.NewEnvironment()
	env.BaseDir = filepath.Dir(path)
	env.SetLoader(evaluator.NewLoader(env.BaseDir))
	if _, err := evaluator.Eval(prog, env); err != nil {
		return FileResult{Path: path, SetupError: err.Error()}
	}
	cases, err := collectTests(prog, env)
	if err != nil {
		return FileResult{Path: path, SetupError: err.Error()}
	}
	result := FileResult{Path: path, Tests: cases}
	for _, tc := range cases {
		if _, err := evaluator.Eval(&ast.CallExpr{Position: tc.Pos, Callee: &ast.Identifier{Position: tc.Pos, Name: tc.Name}}, env); err != nil {
			result.Failures = append(result.Failures, Failure{Name: tc.Name, Pos: tc.Pos, Reason: err.Error()})
		}
	}
	return result
}

func collectTests(prog *ast.Program, env *evaluator.Environment) ([]TestCase, error) {
	cases := []TestCase{}
	seen := map[string]bool{}
	for _, stmt := range prog.Statements {
		name, pos, fn, ok, err := testFromStatement(stmt, env)
		if err != nil {
			return nil, err
		}
		if !ok || seen[name] {
			continue
		}
		seen[name] = true
		cases = append(cases, TestCase{Name: name, Pos: pos, Fn: fn})
	}
	return cases, nil
}

func testFromStatement(stmt ast.Statement, env *evaluator.Environment) (string, ast.Position, *evaluator.Function, bool, error) {
	switch s := stmt.(type) {
	case *ast.VarDecl:
		if s.Value == nil {
			return "", ast.Position{}, nil, false, nil
		}
		lit, ok := s.Value.(*ast.FunctionLiteral)
		if !ok || !isTestFunction(s.Name, lit) {
			return "", ast.Position{}, nil, false, nil
		}
		fn, err := functionFromEnv(s.Name, s.Position, env)
		return s.Name, s.Position, fn, true, err
	case *ast.ExportStmt:
		return testFromStatement(s.Inner, env)
	}
	return "", ast.Position{}, nil, false, nil
}

func isTestFunction(name string, lit *ast.FunctionLiteral) bool {
	if strings.HasPrefix(name, "测试") || strings.HasPrefix(name, "Test") {
		return true
	}
	for _, tag := range lit.Tags {
		if tag.TypeName == "测试" || tag.TypeName == "test" {
			return true
		}
	}
	return false
}

func functionFromEnv(name string, pos ast.Position, env *evaluator.Environment) (*evaluator.Function, error) {
	value, ok := env.Get(name)
	if !ok {
		return nil, fmt.Errorf("测试函数 %s 未定义", name)
	}
	fn, ok := value.(*evaluator.Function)
	if !ok {
		return nil, &evaluator.EvalError{Pos: pos, Reason: fmt.Sprintf("%s 必须是函数", name)}
	}
	if len(fn.Params) != 0 {
		return nil, &evaluator.EvalError{Pos: pos, Reason: fmt.Sprintf("测试函数 %s 不能有参数", name)}
	}
	return fn, nil
}

func (r Runner) print(results []FileResult) {
	if r.Out == nil {
		return
	}
	var passed int
	var failed int
	for _, result := range results {
		displayPath := result.Path
		fmt.Fprintf(r.Out, "%s\n", displayPath)
		if result.SetupError != "" {
			failed++
			fmt.Fprintf(r.Out, "  失败：%s\n", indent(result.SetupError, "       "))
			continue
		}
		for _, tc := range result.Tests {
			failure := findFailure(result.Failures, tc.Name)
			if failure == nil {
				passed++
				fmt.Fprintf(r.Out, "  通过 %s\n", tc.Name)
				continue
			}
			failed++
			fmt.Fprintf(r.Out, "  失败 %s\n", tc.Name)
			fmt.Fprintf(r.Out, "       %s\n", indent(failure.Reason, "       "))
		}
	}
	fmt.Fprintf(r.Out, "\n共 %d 个测试文件，通过 %d 个，失败 %d 个\n", len(results), passed, failed)
}

func findFailure(failures []Failure, name string) *Failure {
	for i := range failures {
		if failures[i].Name == name {
			return &failures[i]
		}
	}
	return nil
}

func indent(text string, prefix string) string {
	lines := strings.Split(text, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

func HasFailure(results []FileResult) bool {
	for _, result := range results {
		if result.Failed() {
			return true
		}
	}
	return false
}
