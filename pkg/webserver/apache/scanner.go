package apache

import "strings"

// scanLogicalLines 把源文本切成逻辑行：合并行尾续行符 \，跳过空行
func scanLogicalLines(src string) []string {
	var out []string
	lines := strings.Split(src, "\n")

	i := 0
	for i < len(lines) {
		raw := strings.TrimRight(lines[i], "\r")
		if strings.TrimSpace(raw) == "" {
			i++
			continue
		}

		// 合并续行：行尾为奇数个反斜杠时，去掉续行符并接上下一行
		buf := raw
		for endsWithOddBackslash(buf) && i+1 < len(lines) {
			buf = buf[:len(buf)-1]
			i++
			buf += strings.TrimRight(lines[i], "\r")
		}

		out = append(out, strings.TrimSpace(buf))
		i++
	}

	return out
}

// endsWithOddBackslash 判断字符串是否以奇数个反斜杠结尾（奇数表示续行符）
func endsWithOddBackslash(s string) bool {
	n := 0
	for i := len(s) - 1; i >= 0 && s[i] == '\\'; i-- {
		n++
	}
	return n%2 == 1
}
