// Package lsp 实现一个最小的 Language Server Protocol（3.17 子集）服务器，
// 通过 stdio 使用 JSON-RPC 与编辑器通信。
package lsp

import "encoding/json"

// Position 是 LSP 的 0-based 位置，字符列以 UTF-16 码元计。
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity"`
	Code     string             `json:"code,omitempty"`
	Source   string             `json:"source,omitempty"`
	Message  string             `json:"message"`
}

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     *int         `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version,omitempty"`
}

type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type didOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type didChangeTextDocumentParams struct {
	TextDocument VersionedTextDocumentIdentifier  `json:"textDocument"`
	Changes      []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type didCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type clientCapabilities struct{}

type initializeParams struct {
	ProcessID    int                `json:"processId"`
	RootURI      string             `json:"rootUri"`
	Capabilities clientCapabilities `json:"capabilities"`
}

type textDocumentSyncOptions struct {
	OpenClose bool `json:"openClose"`
	Change    int  `json:"change"`
}

type serverCapabilities struct {
	TextDocumentSync   textDocumentSyncOptions `json:"textDocumentSync"`
	DefinitionProvider bool                    `json:"definitionProvider"`
	HoverProvider      bool                    `json:"hoverProvider"`
	CompletionProvider *completionOptions      `json:"completionProvider,omitempty"`
	DiagnosticProvider map[string]any          `json:"diagnosticProvider,omitempty"`
}

type completionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters"`
}

type textDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type markupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type hover struct {
	Contents markupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

type lspLocation struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type completionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type completionItem struct {
	Label  string         `json:"label"`
	Kind   int            `json:"kind"`
	Detail string         `json:"detail,omitempty"`
	Doc    *markupContent `json:"documentation,omitempty"`
}

type completionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []completionItem `json:"items"`
}

const (
	completionKindText      = 1
	completionKindFunction  = 3
	completionKindVariable  = 6
	completionKindField     = 5
	completionKindMethod    = 2
	completionKindClass     = 7
	completionKindInterface = 8
	completionKindConstant  = 21
	completionKindParameter = 14
	completionKindModule    = 9
	completionKindKeyword   = 14
)

type initializeResult struct {
	Capabilities serverCapabilities `json:"capabilities"`
	ServerInfo   map[string]string  `json:"serverInfo,omitempty"`
}

type jsonrpcMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *jsonRPCError    `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
