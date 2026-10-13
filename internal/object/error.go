package object

import (
	"fmt"
	"strings"
)

const KindError Kind = "错误"

// StackFrame 记录一次函数调用的源码位置，用于错误调用栈。
type StackFrame struct {
	Function string
	File     string
	Line     int
	Column   int
}

func (f StackFrame) String() string {
	loc := f.File
	if loc == "" {
		loc = "<脚本>"
	}
	if f.Line > 0 {
		return fmt.Sprintf("  由 %s 调用（%s 第%d行）", f.Function, loc, f.Line)
	}
	return fmt.Sprintf("  由 %s 调用（%s）", f.Function, loc)
}

// Error 是凡语言的一等错误值。除消息外还可以携带：
//
//	Code    错误码，便于程序化分支判断
//	Details 上下文字典，存放任意结构化数据
//	Cause   被包装的原始错误，形成原因链
//	Stack   产生错误时的调用栈（由解释器自动填充）
type Error struct {
	Message string
	Code    string
	Details *Dict
	Cause   *Error
	Stack   []StackFrame
}

func (e *Error) Kind() Kind { return KindError }

func (e *Error) Inspect() string {
	var b strings.Builder
	b.WriteString("错误：")
	b.WriteString(e.Message)
	if e.Code != "" {
		b.WriteString("（")
		b.WriteString(e.Code)
		b.WriteString("）")
	}
	return b.String()
}

func (e *Error) Error() string { return e.Message }

func NewError(message string) *Error {
	return &Error{Message: message}
}

// NewErrorf 按格式化方式创建错误。
func NewErrorf(format string, args ...interface{}) *Error {
	return NewError(fmt.Sprintf(format, args...))
}

// WithCode 设置错误码并返回自身。
func (e *Error) WithCode(code string) *Error {
	e.Code = code
	return e
}

// WithDetails 附加结构化上下文字典并返回自身。
func (e *Error) WithDetails(d *Dict) *Error {
	e.Details = d
	return e
}

// WithCause 包装一个原始错误并返回自身。
func (e *Error) WithCause(cause *Error) *Error {
	e.Cause = cause
	return e
}

// Trace 返回“错误 + 原因链 + 调用栈”的完整文本。
func (e *Error) Trace() string {
	var b strings.Builder
	b.WriteString(e.Inspect())
	for cur := e.Cause; cur != nil; cur = cur.Cause {
		b.WriteString("\n原因：")
		b.WriteString(cur.Inspect())
	}
	for _, frame := range e.Stack {
		b.WriteString("\n")
		b.WriteString(frame.String())
	}
	return b.String()
}
