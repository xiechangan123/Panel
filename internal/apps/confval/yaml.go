package confval

import (
	"strings"

	"github.com/spf13/cast"
)

// GetYAML 读取 YAML 配置项，优先平铺键（安装脚本用 sed 生成的格式），回退到嵌套键
func GetYAML(cfg map[string]any, key string) string {
	if val, ok := cfg[key]; ok {
		return cast.ToString(val)
	}

	prefix, rest, nested := strings.Cut(key, ".")
	val, ok := cfg[prefix]
	if !ok {
		return ""
	}
	if !nested {
		return cast.ToString(val)
	}
	child, ok := val.(map[string]any)
	if !ok {
		return ""
	}

	return GetYAML(child, rest)
}

// SetYAML 以平铺键写入 YAML 配置项，同时清理可能残留的同名嵌套键
func SetYAML(cfg map[string]any, key string, value string) {
	if value == "" {
		return
	}

	cfg[key] = value
	prefix, rest, nested := strings.Cut(key, ".")
	if !nested {
		return
	}
	child, ok := cfg[prefix].(map[string]any)
	if !ok {
		return
	}
	delete(child, rest)
	if len(child) == 0 {
		delete(cfg, prefix)
	}
}
