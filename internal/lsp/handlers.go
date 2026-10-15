package lsp

import (
	"encoding/json"

	"fan/internal/analysis"
)

func (s *Server) getDoc(uri string) *document {
	s.docsMu.Lock()
	defer s.docsMu.Unlock()
	return s.docs[uri]
}

func (s *Server) respondHover(id *json.RawMessage, params textDocumentPositionParams) {
	doc := s.getDoc(params.TextDocument.URI)
	if doc == nil || doc.index == nil {
		_ = s.writeMessage(jsonrpcMessage{ID: id, Result: nil})
		return
	}
	result := doc.index.Hover(params.Position.Line, params.Position.Character)
	if result == nil {
		_ = s.writeMessage(jsonrpcMessage{ID: id, Result: nil})
		return
	}
	_ = s.writeMessage(jsonrpcMessage{ID: id, Result: hover{
		Contents: markupContent{Kind: "markdown", Value: result.Content},
		Range:    toLSPRange(result.Range),
	}})
}

func (s *Server) respondDefinition(id *json.RawMessage, params textDocumentPositionParams) {
	doc := s.getDoc(params.TextDocument.URI)
	if doc == nil || doc.index == nil {
		_ = s.writeMessage(jsonrpcMessage{ID: id, Result: nil})
		return
	}
	loc := doc.index.Definition(params.Position.Line, params.Position.Character)
	if loc == nil {
		_ = s.writeMessage(jsonrpcMessage{ID: id, Result: nil})
		return
	}
	_ = s.writeMessage(jsonrpcMessage{ID: id, Result: lspLocation{
		URI: loc.URI, Range: *toLSPRange(loc.Range),
	}})
}

func (s *Server) respondCompletion(id *json.RawMessage, params completionParams) {
	doc := s.getDoc(params.TextDocument.URI)
	items := []completionItem{}
	if doc != nil && doc.index != nil {
		for _, item := range doc.index.Completion(params.Position.Line, params.Position.Character) {
			ci := completionItem{
				Label: item.Label, Kind: toCompletionKind(item.Kind), Detail: item.Detail,
			}
			if item.Doc != "" {
				ci.Doc = &markupContent{Kind: "markdown", Value: item.Doc}
			}
			items = append(items, ci)
		}
	}
	_ = s.writeMessage(jsonrpcMessage{ID: id, Result: completionList{
		IsIncomplete: false, Items: items,
	}})
}

func toLSPRange(r analysis.Range) *Range {
	return &Range{
		Start: Position{Line: r.Start.Line, Character: r.Start.Character},
		End:   Position{Line: r.End.Line, Character: r.End.Character},
	}
}

func toCompletionKind(kind analysis.SymbolKind) int {
	switch kind {
	case analysis.KindFunction, analysis.KindMixfixFunction:
		return completionKindFunction
	case analysis.KindMethod:
		return completionKindMethod
	case analysis.KindClass, analysis.KindInterface:
		return completionKindInterface
	case analysis.KindField:
		return completionKindField
	case analysis.KindVariable, analysis.KindParameter:
		return completionKindVariable
	case analysis.KindConstant:
		return completionKindConstant
	case analysis.KindModule:
		return completionKindModule
	default:
		return completionKindText
	}
}
