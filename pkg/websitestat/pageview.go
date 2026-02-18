package websitestat

import (
	"path"
	"strings"
)

// 静态资源扩展名（非页面浏览）
var staticExts = map[string]struct{}{
	".js": {}, ".css": {}, ".png": {}, ".jpg": {}, ".jpeg": {},
	".gif": {}, ".svg": {}, ".ico": {}, ".woff": {}, ".woff2": {},
	".ttf": {}, ".eot": {}, ".otf": {}, ".map": {}, ".webp": {},
	".avif": {}, ".mp4": {}, ".mp3": {}, ".webm": {}, ".ogg": {},
	".pdf": {}, ".zip": {}, ".gz": {}, ".tar": {}, ".rar": {},
	".7z": {}, ".bz2": {}, ".xz": {}, ".swf": {}, ".flv": {},
}

// IsPageView 判定请求是否为页面浏览（PV）
func IsPageView(entry *LogEntry) bool {
	// 优先使用 Content-Type 判定
	if entry.ContentType != "" {
		return strings.Contains(entry.ContentType, "text/html")
	}

	// 回退到 URI 扩展名判定
	ext := strings.ToLower(path.Ext(entry.URI))
	// 无扩展名视为页面（如 /about, /api/xxx）
	if ext == "" {
		return true
	}
	// 检查是否为已知静态资源
	_, isStatic := staticExts[ext]
	return !isStatic
}
