package confval

import (
	"strconv"
	"strings"

	"github.com/spf13/cast"
)

// tomlCodec TOML 根表的书写约定，值的引号由 formatTOML 按类型决定
var tomlCodec = Codec{Delim: "=", Assign: " = ", Comments: []string{"#"}, InlineComment: true}

// GetTOML 读取根表配置项，兼容点号键（auth.token = "x"）与分组写法（[auth] 下的 token = "x"）
func GetTOML(content, key string) string {
	section, leaf, nested := cutLast(key)

	current := ""
	for raw := range strings.SplitSeq(content, "\n") {
		if name, ok := tableName(raw); ok {
			current = name
			continue
		}

		parsed := tomlCodec.parse(raw)
		if !parsed.ok || parsed.commented != "" {
			continue
		}
		// 根表下写点号全键，分组内写末段键
		if (current == "" && parsed.key == key) || (nested && current == section && parsed.key == leaf) {
			return unquoteTOML(parsed.value)
		}
	}

	return ""
}

// SetTOML 写入根表配置项，值为空时注释掉该项；新键以点号形式插入到首个表头之前
func SetTOML(content, key string, value any) string {
	literal := formatTOML(value)
	section, leaf, nested := cutLast(key)

	raws := strings.Split(content, "\n")
	result := make([]string, 0, len(raws)+1)
	current := ""
	found := false
	// 首个表头在 result 中的位置，新键只能插在它之前才仍属于根表
	firstTable := -1

	for _, raw := range raws {
		if name, ok := tableName(raw); ok {
			current = name
			if firstTable < 0 {
				firstTable = len(result)
			}
			result = append(result, raw)
			continue
		}

		parsed := tomlCodec.parse(raw)
		matched := parsed.ok && ((current == "" && parsed.key == key) || (nested && current == section && parsed.key == leaf))
		if !matched || found {
			result = append(result, raw)
			continue
		}
		found = true

		if literal == "" {
			// 值为空时注释掉该配置项，已注释的原样保留
			if parsed.commented == "" {
				result = append(result, parsed.indent+"# "+strings.TrimSpace(raw))
			} else {
				result = append(result, raw)
			}
			continue
		}

		// 保留原行的写法、缩进与行尾注释，只替换值
		result = append(result, parsed.indent+parsed.key+tomlCodec.Assign+literal+trailing(parsed.comment))
	}

	if found || literal == "" {
		return strings.Join(result, "\n")
	}

	newLine := key + tomlCodec.Assign + literal
	if firstTable < 0 {
		return strings.Join(append(result, newLine), "\n")
	}

	return strings.Join(insertAt(result, firstTable, newLine), "\n")
}

// formatTOML 按 Go 类型输出 TOML 字面量，零值返回空串表示删除该项
func formatTOML(value any) string {
	switch v := value.(type) {
	case string:
		if v == "" {
			return ""
		}
		return strconv.Quote(v)
	case bool:
		return strconv.FormatBool(v)
	case []string:
		if len(v) == 0 {
			return ""
		}
		quoted := make([]string, 0, len(v))
		for _, item := range v {
			quoted = append(quoted, strconv.Quote(item))
		}
		return "[" + strings.Join(quoted, ", ") + "]"
	default:
		return cast.ToString(v)
	}
}

// unquoteTOML 去除字符串值的引号，非字符串值原样返回
func unquoteTOML(value string) string {
	if unquoted, err := strconv.Unquote(value); err == nil {
		return unquoted
	}

	return value
}

// tableName 解析 [table] 与 [[table]] 行
func tableName(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return "", false
	}

	return strings.Trim(strings.TrimSpace(strings.Trim(trimmed, "[]")), " \t"), true
}

// cutLast 切出点号键的分组前缀与末段，如 auth.token -> auth, token
func cutLast(key string) (section, leaf string, nested bool) {
	idx := strings.LastIndex(key, ".")
	if idx < 0 {
		return "", key, false
	}

	return key[:idx], key[idx+1:], true
}
