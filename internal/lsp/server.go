package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"fan/internal/analysis"
)

// Server 是凡语言 LSP 服务器。
type Server struct {
	in  *bufio.Reader
	out io.Writer

	docsMu sync.Mutex
	docs   map[string]*document

	writeMu sync.Mutex
	stop    chan struct{}
}

type document struct {
	uri     string
	version int
	text    string
	index   *analysis.Index
}

// NewServer 创建一个从 in 读请求、向 out 写响应的 LSP 服务器。
func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{
		in:   bufio.NewReader(in),
		out:  out,
		docs: map[string]*document{},
		stop: make(chan struct{}),
	}
}

// Run 处理连接直到收到 exit 请求或输入结束。
func (s *Server) Run() error {
	defer close(s.stop)
	for {
		msg, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if msg.Method == "exit" {
			return nil
		}
		s.handle(msg)
	}
}

func (s *Server) readMessage() (jsonrpcMessage, error) {
	var contentLength int
	for {
		line, err := s.in.ReadString('\n')
		if err != nil {
			return jsonrpcMessage{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if value, ok := strings.CutPrefix(line, "Content-Length:"); ok {
			contentLength, _ = strconv.Atoi(strings.TrimSpace(value))
		}
	}
	if contentLength <= 0 {
		return jsonrpcMessage{}, fmt.Errorf("缺少 Content-Length 头")
	}
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(s.in, body); err != nil {
		return jsonrpcMessage{}, err
	}
	var msg jsonrpcMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return jsonrpcMessage{}, err
	}
	return msg, nil
}

func (s *Server) writeMessage(msg jsonrpcMessage) error {
	msg.JSONRPC = "2.0"
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, err := fmt.Fprintf(s.out, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = s.out.Write(body)
	return err
}

func (s *Server) handle(msg jsonrpcMessage) {
	switch msg.Method {
	case "initialize":
		var params initializeParams
		_ = json.Unmarshal(msg.Params, &params)
		_ = s.writeMessage(jsonrpcMessage{
			ID: msg.ID,
			Result: initializeResult{
				Capabilities: serverCapabilities{
					TextDocumentSync:   textDocumentSyncOptions{OpenClose: true, Change: 1},
					DefinitionProvider: true,
					HoverProvider:      true,
					CompletionProvider: &completionOptions{TriggerCharacters: []string{".", "的"}},
				},
				ServerInfo: map[string]string{"name": "fan-lsp", "version": "0.1.0"},
			},
		})
	case "initialized", "$/setTrace", "$/cancelRequest":
	case "shutdown":
		_ = s.writeMessage(jsonrpcMessage{ID: msg.ID, Result: nil})
	case "textDocument/didOpen":
		var params didOpenTextDocumentParams
		if json.Unmarshal(msg.Params, &params) == nil {
			s.setDocument(params.TextDocument.URI, params.TextDocument.Version, params.TextDocument.Text)
			s.publish(params.TextDocument.URI, params.TextDocument.Version)
		}
	case "textDocument/didChange":
		var params didChangeTextDocumentParams
		if json.Unmarshal(msg.Params, &params) == nil && len(params.Changes) > 0 {
			text := params.Changes[len(params.Changes)-1].Text
			s.setDocument(params.TextDocument.URI, params.TextDocument.Version, text)
			s.schedulePublish(params.TextDocument.URI, params.TextDocument.Version)
		}
	case "textDocument/didSave":
		var params struct {
			TextDocument TextDocumentIdentifier `json:"textDocument"`
		}
		if json.Unmarshal(msg.Params, &params) == nil {
			s.schedulePublish(params.TextDocument.URI, 0)
		}
	case "textDocument/didClose":
		var params didCloseTextDocumentParams
		if json.Unmarshal(msg.Params, &params) == nil {
			s.removeDocument(params.TextDocument.URI)
			_ = s.writeMessage(jsonrpcMessage{
				Method: "textDocument/publishDiagnostics",
				Params: mustMarshal(PublishDiagnosticsParams{
					URI: params.TextDocument.URI, Diagnostics: []Diagnostic{},
				}),
			})
		}
	case "textDocument/hover":
		var params textDocumentPositionParams
		if json.Unmarshal(msg.Params, &params) == nil {
			s.respondHover(msg.ID, params)
		}
	case "textDocument/definition":
		var params textDocumentPositionParams
		if json.Unmarshal(msg.Params, &params) == nil {
			s.respondDefinition(msg.ID, params)
		}
	case "textDocument/completion":
		var params completionParams
		if json.Unmarshal(msg.Params, &params) == nil {
			s.respondCompletion(msg.ID, params)
		}
	default:
		if msg.ID != nil {
			_ = s.writeMessage(jsonrpcMessage{
				ID:    msg.ID,
				Error: &jsonRPCError{Code: -32601, Message: "方法未实现：" + msg.Method},
			})
		}
	}
}

func mustMarshal(value any) json.RawMessage {
	body, _ := json.Marshal(value)
	return body
}

func (s *Server) setDocument(uri string, version int, text string) {
	s.docsMu.Lock()
	defer s.docsMu.Unlock()
	s.docs[uri] = &document{
		uri: uri, version: version, text: text,
		index: analysis.BuildIndex(uri, text),
	}
}

func (s *Server) removeDocument(uri string) {
	s.docsMu.Lock()
	defer s.docsMu.Unlock()
	delete(s.docs, uri)
}

// schedulePublish 做简单防抖：停顿 250ms 后推送最新诊断。
func (s *Server) schedulePublish(uri string, version int) {
	go func() {
		timer := time.NewTimer(250 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-timer.C:
			s.publish(uri, version)
		case <-s.stop:
		}
	}()
}

func (s *Server) publish(uri string, version int) {
	s.docsMu.Lock()
	doc, ok := s.docs[uri]
	if ok {
		text := doc.text
		index := doc.index
		if version == 0 {
			version = doc.version
		}
		s.docsMu.Unlock()
		var rawDiags []analysis.Diagnostic
		if index != nil {
			rawDiags = analysis.AnalyzeIndex(index)
		} else {
			rawDiags = analysis.Analyze(text)
		}
		diagnostics := toLSPDiagnostics(text, rawDiags)
		var versionPtr *int
		if version != 0 {
			v := version
			versionPtr = &v
		}
		_ = s.writeMessage(jsonrpcMessage{
			Method: "textDocument/publishDiagnostics",
			Params: mustMarshal(PublishDiagnosticsParams{
				URI: uri, Version: versionPtr, Diagnostics: diagnostics,
			}),
		})
		return
	}
	s.docsMu.Unlock()
}

// toLSPDiagnostics 把分析诊断转换为 LSP 诊断，列号转换为 UTF-16 码元。
func toLSPDiagnostics(source string, diags []analysis.Diagnostic) []Diagnostic {
	lines := strings.Split(source, "\n")
	result := make([]Diagnostic, 0, len(diags))
	for _, d := range diags {
		lineText := ""
		if d.Line-1 >= 0 && d.Line-1 < len(lines) {
			lineText = lines[d.Line-1]
		}
		startChar := analysis.UTF16Column(lineText, d.Column)
		endChar := analysis.UTF16Column(lineText, d.EndCol)
		if endChar < startChar {
			endChar = startChar + 1
		}
		severity := SeverityWarning
		switch d.Severity {
		case analysis.SeverityError:
			severity = SeverityError
		case analysis.SeverityWarning:
			severity = SeverityWarning
		case analysis.SeverityInformation:
			severity = SeverityInformation
		case analysis.SeverityHint:
			severity = SeverityHint
		}
		result = append(result, Diagnostic{
			Range: Range{
				Start: Position{Line: d.Line - 1, Character: startChar},
				End:   Position{Line: d.EndLine - 1, Character: endChar},
			},
			Severity: severity,
			Code:     d.Code,
			Source:   "fan",
			Message:  d.Message,
		})
	}
	return result
}
