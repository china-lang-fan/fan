package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fan/internal/evaluator"
	"fan/internal/object"
	"fan/internal/parser"
	"fan/internal/testrun"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "错误：", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	switch args[0] {
	case "version", "-v", "--version":
		fmt.Println("凡语言", version, "（命令名：fan）")
		return nil
	case "run":
		if len(args) < 2 {
			return fmt.Errorf("用法：fan run <脚本.fan>")
		}
		return runFile(args[1])
	case "test":
		target := "."
		if len(args) >= 2 {
			target = args[1]
		}
		results, err := testrun.Runner{Out: os.Stdout}.Run(target)
		if err != nil {
			return err
		}
		if testrun.HasFailure(results) {
			return fmt.Errorf("测试失败")
		}
		return nil
	case "repl":
		return runREPL(os.Stdin, os.Stdout)
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("未知子命令：%s", args[0])
	}
}

func runFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	prog, errs := parser.ParseProgram(string(data))
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "语法错误：", e.Error())
		}
		return fmt.Errorf("解析失败")
	}
	dir := filepath.Dir(absPath(path))
	env := evaluator.NewEnvironment()
	env.BaseDir = dir
	env.Loader = evaluator.NewLoader(dir)
	if _, err := evaluator.Eval(prog, env); err != nil {
		return err
	}
	return nil
}

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func runREPL(in io.Reader, out io.Writer) error {
	env := evaluator.NewEnvironment()
	scanner := bufio.NewScanner(in)
	fmt.Fprintln(out, "凡语言 REPL，输入 :退出 结束")
	for {
		fmt.Fprint(out, "fan> ")
		if !scanner.Scan() {
			fmt.Fprintln(out)
			return scanner.Err()
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == ":退出" || line == ":quit" || line == ":q" {
			return nil
		}
		prog, errs := parser.ParseProgram(line)
		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintln(out, "语法错误：", e.Error())
			}
			continue
		}
		res, err := evaluator.Eval(prog, env)
		if err != nil {
			fmt.Fprintln(out, "运行错误：", err.Error())
			continue
		}
		if res != nil {
			fmt.Fprintln(out, object.Format(res))
		}
	}
}

func printUsage() {
	fmt.Println("凡语言 —— 中文通用脚本语言（命令名：fan）")
	fmt.Println("用法：")
	fmt.Println("  fan run <脚本.fan>   运行脚本")
	fmt.Println("  fan test [路径]      运行测试")
	fmt.Println("  fan repl             进入交互模式")
	fmt.Println("  fan version          显示版本")
	fmt.Println("  fan help             显示帮助")
}
