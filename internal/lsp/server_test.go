package lsp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type frames struct {
	messages []jsonrpcMessage
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
	initParams := map[string]any{
		"processId":    1,
		"rootUri":      nil,
		"capabilities": map[string]any{},
	}
	input.WriteString(frame("initialize", 1, initParams))
	input.WriteString(frame("initialized", nil, map[string]any{}))
	input.WriteString(frame("textDocument/didOpen", nil, map[string]any{
		"textDocument": map[string]any{
			"uri": "file:///test.凡", "languageId": "fan", "version": 1, "text": source,
		},
	}))

	var output bytes.Buffer
	server := NewServer(&input, &output)
	done := make(chan error, 1)
	go func() { done <- server.Run() }()

	var messages []jsonrpcMessage
	deadline := time.After(2 * time.Second)
	for {
		if strings.Contains(output.String(), "mixfix-space") {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("等待诊断超时，输出：%s", output.String())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	messages = readFrames(t, output.String())
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("服务器异常：%v", err)
		}
	default:
	}
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
