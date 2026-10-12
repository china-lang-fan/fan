package analysis

import (
	"os"
	"path/filepath"
	"strings"
)

// Index 跨文件符号索引：以主文件为入口，静态解析其导入的本地模块。
// SDK 模块（可执行文件旁的 sdk 目录等）同样只 parse 不执行。
type Index struct {
	root       *Document
	baseDir    string
	docs       map[string]*Document // 绝对路径（去扩展名后规范化）-> 文档
	searchDirs []string
}

// BuildIndex 解析入口源码，并静态加载它导入的模块。
// uri 可以是 file:// URI 或文件路径；找不到文件时仍返回仅含入口的索引。
func BuildIndex(uri, source string) *Index {
	path := uriToPath(uri)
	base := filepath.Dir(path)
	index := &Index{
		baseDir:    base,
		docs:       map[string]*Document{},
		searchDirs: moduleSearchDirs(base),
	}
	index.root = ParseDocument(uri, source)
	if path != "" {
		index.docs[canonicalPath(path)] = index.root
		index.loadImports(index.root, map[string]bool{})
	}
	return index
}

// Root 返回入口文档。
func (idx *Index) Root() *Document { return idx.root }

func (idx *Index) loadImports(doc *Document, visited map[string]bool) {
	for _, imp := range doc.imports {
		full, ok := idx.resolve(imp.path)
		if !ok {
			continue
		}
		key := canonicalPath(full)
		if visited[key] {
			continue
		}
		visited[key] = true
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		moduleDoc := ParseDocument(pathToURI(full), string(data))
		idx.docs[key] = moduleDoc
		idx.loadImports(moduleDoc, visited)
	}
}

func (idx *Index) resolve(importPath string) (string, bool) {
	candidates := []string{importPath}
	if !strings.HasSuffix(importPath, ".fan") && !strings.HasSuffix(importPath, ".凡") {
		candidates = []string{importPath + ".凡", importPath + ".fan", importPath}
	}
	try := func(p string) (string, bool) {
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			return "", false
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return "", false
		}
		return abs, true
	}
	if filepath.IsAbs(importPath) {
		for _, c := range candidates {
			if full, ok := try(c); ok {
				return full, true
			}
		}
		return "", false
	}
	for _, dir := range idx.searchDirs {
		for _, c := range candidates {
			if full, ok := try(filepath.Join(dir, c)); ok {
				return full, true
			}
		}
	}
	for _, c := range candidates {
		if full, ok := try(c); ok {
			return full, true
		}
	}
	return "", false
}

func moduleSearchDirs(base string) []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(dir string) {
		if dir == "" {
			return
		}
		abs, err := filepath.Abs(dir)
		if err != nil || seen[abs] {
			return
		}
		seen[abs] = true
		dirs = append(dirs, abs)
	}
	add(base)
	if executable, err := os.Executable(); err == nil {
		execDir := filepath.Dir(executable)
		add(execDir)
		add(filepath.Join(execDir, "sdk"))
		add(filepath.Join(execDir, "..", "share", "fan", "sdk"))
		add(filepath.Join(execDir, "..", "..", "share", "fan", "sdk"))
	}
	add("/usr/local/share/fan/sdk")
	add("/usr/share/fan/sdk")
	return dirs
}

func canonicalPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
