package lsp

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// safeBuffer 是带锁的输出缓冲，供测试在服务器写入时安全读取。
type safeBuffer struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	cond chan struct{}
}

func newSafeBuffer() *safeBuffer {
	return &safeBuffer{cond: make(chan struct{}, 256)}
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	n, err := b.buf.Write(p)
	b.mu.Unlock()
	select {
	case b.cond <- struct{}{}:
	default:
	}
	return n, err
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func frame(method string, id any, params any) string {
	var idRaw *json.RawMessage
	if id != nil {
		body, _ := json.Marshal(id)
		raw := json.RawMessage(body)
		idRaw = &raw
	}
	var paramsRaw json.RawMessage
	if params != nil {
		paramsRaw, _ = json.Marshal(params)
	}
	msg := jsonrpcMessage{JSONRPC: "2.0", ID: idRaw, Method: method, Params: paramsRaw}
	body, _ := json.Marshal(msg)
	return "Content-Length: " + itoa(len(body)) + "\r\n\r\n" + string(body)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func readFrames(t *testing.T, data string) []jsonrpcMessage {
	t.Helper()
	var result []jsonrpcMessage
	rest := data
	for {
		idx := strings.Index(rest, "\r\n\r\n")
		if idx < 0 {
			break
		}
		header := rest[:idx]
		var length int
		for _, line := range strings.Split(header, "\r\n") {
			if value, ok := strings.CutPrefix(line, "Content-Length:"); ok {
				for _, r := range strings.TrimSpace(value) {
					length = length*10 + int(r-'0')
				}
			}
		}
		if idx+4+length > len(rest) {
			break
		}
		body := rest[idx+4 : idx+4+length]
		var msg jsonrpcMessage
		if err := json.Unmarshal([]byte(body), &msg); err != nil {
			t.Fatalf("解析响应帧失败：%v", err)
		}
		result = append(result, msg)
		rest = rest[idx+4+length:]
	}
	return result
}

func TestLSPInitializeAndDiagnostics(t *testing.T) {
	source := "函数 修改(值 字符串)的状态为无效\n    返回 值\n结束\n\n修改(\"x\") 的状态为无效\n"
	var input bytes.Buffer
	input.WriteString(frame("initialize", 1, map[string]any{
		"processId":    1,
		"rootUri":      nil,
		"capabilities": map[string]any{},
	}))
	input.WriteString(frame("initialized", nil, map[string]any{}))
	input.WriteString(frame("textDocument/didOpen", nil, map[string]any{
		"textDocument": map[string]any{
			"uri": "file:///test.凡", "languageId": "fan", "version": 1, "text": source,
		},
	}))
	input.WriteString(frame("exit", nil, nil))

	output := newSafeBuffer()
	server := NewServer(&input, output)
	done := make(chan error, 1)
	go func() { done <- server.Run() }()

	// didOpen 同步推送诊断，收到输出即说明已处理；再发 exit 让服务器干净退出。
	deadline := time.After(2 * time.Second)
	for {
		if strings.Contains(output.String(), "mixfix-space") {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("等待诊断超时，输出：%s", output.String())
		case <-time.After(5 * time.Millisecond):
		}
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("服务器异常：%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("服务器未在 exit 后退出")
	}

	messages := readFrames(t, output.String())
	foundInit := false
	foundDiag := false
	for _, msg := range messages {
		if msg.ID != nil && msg.Method == "" && msg.Error == nil {
			foundInit = true
		}
		if msg.Method == "textDocument/publishDiagnostics" {
			var params PublishDiagnosticsParams
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				t.Fatal(err)
			}
			for _, d := range params.Diagnostics {
				if d.Code == "mixfix-space" {
					foundDiag = true
				}
			}
		}
	}
	if !foundInit {
		t.Fatal("未收到 initialize 响应")
	}
	if !foundDiag {
		t.Fatal("未收到 mixfix-space 诊断")
	}
}

func TestLSPCapabilitiesAndHover(t *testing.T) {
	source := "# 打招呼函数\n函数 打招呼(名字 字符串) -> 字符串\n    返回 名字\n结束\n\n变量 x = 打招呼(\"张三\")\n"
	var input bytes.Buffer
	input.WriteString(frame("initialize", 1, map[string]any{"capabilities": map[string]any{}}))
	input.WriteString(frame("initialized", nil, map[string]any{}))
	input.WriteString(frame("textDocument/didOpen", nil, map[string]any{
		"textDocument": map[string]any{
			"uri": "file:///t.凡", "languageId": "fan", "version": 1, "text": source,
		},
	}))
	input.WriteString(frame("textDocument/hover", 2, map[string]any{
		"textDocument": map[string]any{"uri": "file:///t.凡"},
		"position":     map[string]any{"line": 5, "character": 7},
	}))
	input.WriteString(frame("exit", nil, nil))

	output := newSafeBuffer()
	server := NewServer(&input, output)
	if err := server.Run(); err != nil {
		t.Fatal(err)
	}
	var hoverFound bool
	for _, msg := range readFrames(t, output.String()) {
		if msg.ID != nil {
			var id int
			_ = json.Unmarshal(*msg.ID, &id)
			if id == 2 && msg.Method == "" {
				var payload struct {
					Contents struct{ Value string } `json:"contents"`
				}
				raw, _ := json.Marshal(msg.Result)
				_ = json.Unmarshal(raw, &payload)
				if strings.Contains(payload.Contents.Value, "打招呼函数") &&
					strings.Contains(payload.Contents.Value, "打招呼") {
					hoverFound = true
				}
			}
		}
	}
	if !hoverFound {
		t.Fatalf("悬停未返回函数签名和文档注释：%s", output.String())
	}
}
