package types

import "strings"

// LocationKind location 的匹配方式，面板统一用 nginx 的写法表达，各方言自行渲染
type LocationKind uint8

const (
	LocationPrefix   LocationKind = iota // 前缀匹配，含 ^~
	LocationExact                        // = 精确匹配
	LocationRegex                        // ~ 正则
	LocationRegexAny                     // ~* 正则，不区分大小写
)

// ParseLocation 拆出匹配方式与其后的路径或正则。nginx 允许操作符与后串之间没有空格
func ParseLocation(location string) (LocationKind, string) {
	location = strings.TrimSpace(location)
	for _, op := range []struct {
		prefix string
		kind   LocationKind
	}{
		{"~*", LocationRegexAny},
		{"~", LocationRegex},
		{"=", LocationExact},
		{"^~", LocationPrefix},
	} {
		if rest, ok := strings.CutPrefix(location, op.prefix); ok {
			return op.kind, strings.TrimSpace(rest)
		}
	}

	return LocationPrefix, location
}

// NormalizePath 补上开头的斜杠
func NormalizePath(path string) string {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}
