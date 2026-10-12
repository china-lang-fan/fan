package analysis

import (
	"net/url"
	"path/filepath"
	"strings"
)

// uriToPath 把 file:// URI 转换为本地路径；非 URI 输入原样返回。
func uriToPath(uri string) string {
	if strings.HasPrefix(uri, "file://") {
		if u, err := url.Parse(uri); err == nil {
			return u.Path
		}
	}
	return uri
}

// pathToURI 把本地路径转换为 file:// URI。
func pathToURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "file://" + path
	}
	return "file://" + abs
}
