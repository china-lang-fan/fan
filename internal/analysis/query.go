package analysis

import (
	"strings"

	"fan/internal/ast"
)

// Location 指向某个文件中的源码区域。
type Location struct {
	URI   string
	Range Range
}

// Range 是 0-based、UTF-16 列的区间，与 LSP 对齐。
type Range struct {
	Start Position
	End   Position
}

// Position 是 0-based 位置。
type Position struct {
	Line      int
	Character int
}

// HoverResult 是悬停查询结果。
type HoverResult struct {
	Content string
	Range   Range
}

// CompletionItem 是一个补全候选项。
type CompletionItem struct {
	Label  string
	Kind   SymbolKind
	Detail string
	Doc    string
}

// WordAt 返回指定 0-based UTF-16 位置上的标识符词及其在源码中的 rune 区间。
type wordMatch struct {
	text     string
	startCol int // 1-based
	endCol   int // 1-based，闭区间后一位
}

func (d *Document) wordAt(line0, char16 int) wordMatch {
	if line0 < 0 || line0 >= len(d.lines) {
		return wordMatch{}
	}
	line := d.lines[line0]
	runes := []rune(line)
	// UTF-16 列转 rune 列（本语言标识符都是 BMP 字符，常规情况 1:1；
	// 遇到代理对字符时做折算）。
	runeCol := utf16ToRuneCol(line, char16)
	if runeCol < 0 || runeCol > len(runes) {
		return wordMatch{}
	}
	_ = isLetter
	if runeCol < len(runes) && !isLetter(runes[runeCol]) && runeCol > 0 && isLetter(runes[runeCol-1]) {
		runeCol--
	}
	start := runeCol
	for start > 0 && isLetter(runes[start-1]) {
		start--
	}
	end := runeCol
	for end < len(runes) && isLetter(runes[end]) {
		end++
	}
	if start == end {
		return wordMatch{}
	}
	return wordMatch{text: string(runes[start:end]), startCol: start + 1, endCol: end + 1}
}

func isLetter(r rune) bool {
	return r == '_' ||
		(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		r >= 0x4E00 && r <= 0x9FFF ||
		r >= 0x3400 && r <= 0x4DBF
}

func utf16ToRuneCol(line string, utf16col int) int {
	runes := []rune(line)
	units := 0
	for i, r := range runes {
		if units >= utf16col {
			return i
		}
		if r > 0xFFFF {
			units += 2
		} else {
			units++
		}
	}
	return len(runes)
}

func symbolRange(doc *Document, sym *Symbol) Range {
	return Range{
		Start: Position{Line: sym.Pos.Line - 1, Character: UTF16Column(doc.lineText(sym.Pos.Line), sym.Pos.Column)},
		End:   Position{Line: sym.EndPos.Line - 1, Character: UTF16Column(doc.lineText(sym.EndPos.Line), sym.EndPos.Column)},
	}
}

func nameRange(doc *Document, name string, pos ast.Position) Range {
	line := doc.lineText(pos.Line)
	col := indexOfName(line, name, pos.Column)
	start := Position{Line: pos.Line - 1, Character: UTF16Column(line, col)}
	end := Position{Line: pos.Line - 1, Character: UTF16Column(line, col+len([]rune(name)))}
	return Range{Start: start, End: end}
}

// indexOfName 在一行内从声明起始列附近找到名字所在列。
func indexOfName(line, name string, fallback int) int {
	idx := strings.Index(line, name)
	if idx < 0 {
		return fallback
	}
	return len([]rune(line[:idx])) + 1
}

// Hover 查询入口文档指定位置符号的说明。
func (idx *Index) Hover(line0, char16 int) *HoverResult {
	doc := idx.root
	word := doc.wordAt(line0, char16)
	if word.text == "" {
		return nil
	}
	rng := Range{
		Start: Position{Line: line0, Character: UTF16Column(doc.lineText(line0+1), word.startCol)},
		End:   Position{Line: line0, Character: UTF16Column(doc.lineText(line0+1), word.endCol)},
	}
	if sym := idx.lookupRoot(word.text); sym != nil {
		return &HoverResult{Content: formatHover(sym), Range: rng}
	}
	if owner, member, ok := idx.memberAt(line0, word.startCol, char16); ok {
		if sym := idx.lookupMember(owner, member); sym != nil {
			return &HoverResult{Content: formatHover(sym), Range: rng}
		}
	}
	return nil
}

func formatHover(sym *Symbol) string {
	var builder strings.Builder
	if sym.Detail != "" {
		builder.WriteString("```fan\n" + sym.Detail + "\n```")
	}
	if sym.Doc != "" {
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(sym.Doc)
	}
	if builder.Len() == 0 {
		builder.WriteString(sym.Name)
	}
	return builder.String()
}

func (idx *Index) lookupRoot(name string) *Symbol {
	syms := idx.root.byName[name]
	if len(syms) > 0 {
		return syms[len(syms)-1]
	}
	return nil
}

func (idx *Index) lookupMember(owner, member string) *Symbol {
	if imp, ok := idx.root.imports[owner]; ok {
		if modDoc := idx.moduleDoc(imp.path); modDoc != nil {
			if syms := modDoc.byName[member]; len(syms) > 0 {
				return syms[len(syms)-1]
			}
		}
		return nil
	}
	if class := idx.root.classes[owner]; class != nil {
		for _, child := range class.Children {
			if child.Name == member {
				return child
			}
		}
	}
	for _, sym := range idx.root.byName[member] {
		if sym.Kind == KindMethod && sym.Owner == owner {
			return sym
		}
	}
	return nil
}

func (idx *Index) moduleDoc(importPath string) *Document {
	if full, ok := idx.resolve(importPath); ok {
		return idx.docs[canonicalPath(full)]
	}
	return nil
}

// Definition 返回入口文档指定位置符号的定义位置。
func (idx *Index) Definition(line0, char16 int) *Location {
	doc := idx.root
	word := doc.wordAt(line0, char16)
	if word.text == "" {
		return nil
	}
	if sym := idx.lookupRoot(word.text); sym != nil {
		targetDoc := doc
		return &Location{URI: targetDoc.URI, Range: nameRange(targetDoc, sym.Name, sym.Pos)}
	}
	if owner, member, ok := idx.memberAt(line0, word.startCol, char16); ok {
		return idx.memberDefinition(owner, member)
	}
	return nil
}

func (idx *Index) memberDefinition(owner, member string) *Location {
	if imp, ok := idx.root.imports[owner]; ok {
		if modDoc := idx.moduleDoc(imp.path); modDoc != nil {
			if syms := modDoc.byName[member]; len(syms) > 0 {
				sym := syms[len(syms)-1]
				return &Location{URI: modDoc.URI, Range: nameRange(modDoc, sym.Name, sym.Pos)}
			}
		}
		return nil
	}
	class := idx.root.classes[owner]
	if class == nil {
		return nil
	}
	for _, child := range class.Children {
		if child.Name != member {
			continue
		}
		if child.Pos.Line != 0 {
			return &Location{URI: idx.root.URI, Range: nameRange(idx.root, child.Name, child.Pos)}
		}
	}
	for _, sym := range idx.root.byName[member] {
		if sym.Kind == KindMethod && sym.Owner == owner {
			return &Location{URI: idx.root.URI, Range: nameRange(idx.root, sym.Name, sym.Pos)}
		}
	}
	return nil
}

// memberAt 判断光标处标识符是否为“对象.成员/对象 的 成员”的右半部分，
// 返回推断出的属主类型或模块名与成员名。
func (idx *Index) memberAt(line0, memberCol1, char16 int) (string, string, bool) {
	_ = char16
	doc := idx.root
	line := doc.lineText(line0 + 1)
	runes := []rune(line)
	pos := memberCol1 - 1
	i := pos - 1
	for i >= 0 && (runes[i] == ' ' || runes[i] == '\t') {
		i--
	}
	if i < 0 {
		return "", "", false
	}
	var connectorLen int
	if runes[i] == '.' {
		connectorLen = 1
	} else if i >= 2 && string(runes[i-2:i+1]) == "的" {
		connectorLen = 3
	} else {
		return "", "", false
	}
	i -= connectorLen
	for i >= 0 && (runes[i] == ' ' || runes[i] == '\t') {
		i--
	}
	start := i + 1
	for start > 0 && isLetter(runes[start-1]) {
		start--
	}
	if start > i {
		return "", "", false
	}
	owner := string(runes[start : i+1])
	word := doc.wordAt(line0, UTF16Column(line, memberCol1))
	if word.text == "" {
		return "", "", false
	}
	return idx.resolveOwnerType(owner), word.text, true
}

// resolveOwnerType 把表达式名字推断为属主类型/模块名：
// 模块导入名原样返回；若是局部变量，查其声明类型对应的模型名。
func (idx *Index) resolveOwnerType(name string) string {
	if _, ok := idx.root.imports[name]; ok {
		return name
	}
	if t, ok := idx.localType(name); ok {
		return string(t)
	}
	return name
}

// localType 由局部作用域信息推断变量声明类型，第一版仅支持模型名类型。
func (idx *Index) localType(name string) (ast.DeclType, bool) {
	if _, ok := idx.root.classes[name]; ok {
		return ast.DeclType(name), true
	}
	if syms := idx.root.byName[name]; len(syms) > 0 {
		return ast.DeclType(syms[len(syms)-1].Detail), true
	}
	return ast.TypeAny, false
}

// Completion 返回指定位置的补全项。
func (idx *Index) Completion(line0, char16 int) []CompletionItem {
	doc := idx.root
	line := doc.lineText(line0 + 1)
	if owner, ok := completionOwner(line, char16); ok {
		return idx.memberCompletions(owner)
	}
	return idx.rootCompletions()
}

// completionOwner 检查光标前是否为 “.” 或 “的 ”，返回左表达式名字。
func completionOwner(line string, char16 int) (string, bool) {
	runes := []rune(line)
	col := utf16ToRuneCol(line, char16)
	i := col - 1
	for i >= 0 && (runes[i] == ' ' || runes[i] == '\t') {
		i--
	}
	if i < 0 {
		return "", false
	}
	if runes[i] == '.' {
		// 继续向左取标识符
	} else if i >= 2 && string(runes[i-2:i+1]) == "的" {
		i -= 2
	} else {
		return "", false
	}
	j := i - 1
	for j >= 0 && (runes[j] == ' ' || runes[j] == '\t') {
		j--
	}
	start := j + 1
	for start > 0 && isLetter(runes[start-1]) {
		start--
	}
	if start > j {
		return "", false
	}
	return string(runes[start : j+1]), true
}

func (idx *Index) rootCompletions() []CompletionItem {
	seen := map[string]bool{}
	var items []CompletionItem
	add := func(item CompletionItem) {
		if seen[item.Label] {
			return
		}
		seen[item.Label] = true
		items = append(items, item)
	}
	for _, sym := range idx.root.topLevels {
		add(CompletionItem{Label: sym.Name, Kind: sym.Kind, Detail: sym.Detail, Doc: sym.Doc})
	}
	for name, imp := range idx.root.imports {
		add(CompletionItem{Label: name, Kind: KindModule, Detail: imp.path})
		if modDoc := idx.moduleDoc(imp.path); modDoc != nil {
			for _, sym := range modDoc.topLevels {
				add(CompletionItem{Label: sym.Name, Kind: sym.Kind, Detail: sym.Detail, Doc: sym.Doc})
			}
		}
	}
	for _, kw := range CompletionKeywords {
		add(CompletionItem{Label: kw, Kind: -1, Detail: "关键字"})
	}
	return items
}

func (idx *Index) memberCompletions(ownerExpr string) []CompletionItem {
	owner := idx.resolveOwnerType(ownerExpr)
	var items []CompletionItem
	seen := map[string]bool{}
	add := func(item CompletionItem) {
		if seen[item.Label] {
			return
		}
		seen[item.Label] = true
		items = append(items, item)
	}
	if imp, ok := idx.root.imports[owner]; ok {
		if modDoc := idx.moduleDoc(imp.path); modDoc != nil {
			for _, sym := range modDoc.topLevels {
				add(CompletionItem{Label: sym.Name, Kind: sym.Kind, Detail: sym.Detail, Doc: sym.Doc})
			}
		}
		return items
	}
	if class := idx.root.classes[owner]; class != nil {
		for _, child := range class.Children {
			add(CompletionItem{Label: child.Name, Kind: child.Kind, Detail: child.Detail, Doc: child.Doc})
		}
	}
	for _, sym := range idx.root.topLevels {
		if sym.Kind == KindMethod && sym.Owner == owner {
			add(CompletionItem{Label: sym.Name, Kind: sym.Kind, Detail: sym.Detail, Doc: sym.Doc})
		}
	}
	return items
}

// CompletionKeywords 是补全列表中的关键字。
var CompletionKeywords = []string{
	"函数", "变量", "常量", "返回", "如果", "那么", "否则如果", "否则", "结束",
	"当", "循环", "重复", "直到", "遍历", "中的", "跳出", "继续",
	"判断", "当", "其他", "模型", "类", "嵌入", "方法", "自己",
	"导入", "导出", "作为", "稍后", "抛出", "检查", "捕获",
	"整数", "小数", "字符串", "布尔", "数组", "字典", "真", "假", "空",
}
