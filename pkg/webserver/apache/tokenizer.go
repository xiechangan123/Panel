package apache

import "strings"

// tokenizeLine 把单行文本按 shell 风格分词为参数序列
// 空白分隔，支持双引号/单引号，双引号内处理转义
func tokenizeLine(s string) []Argument {
	var args []Argument
	i, n := 0, len(s)

	for i < n {
		// 跳过空白
		for i < n && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= n {
			break
		}

		switch s[i] {
		case '"':
			val, ni := readQuoted(s, i+1, '"', true)
			args = append(args, Argument{Value: val, Quote: QuoteDouble})
			i = ni
		case '\'':
			val, ni := readQuoted(s, i+1, '\'', false)
			args = append(args, Argument{Value: val, Quote: QuoteSingle})
			i = ni
		default:
			val, ni := readBareWord(s, i)
			args = append(args, Argument{Value: val, Quote: QuoteNone})
			i = ni
		}
	}

	return args
}

// readQuoted 从 start（引号内首字符）读到匹配的结束引号
// unescape 为 true 时按双引号语义处理 \" 与 \\ 转义；返回解引号值与结束后位置
func readQuoted(s string, start int, quote byte, unescape bool) (string, int) {
	var b strings.Builder
	i, n := start, len(s)

	for i < n {
		c := s[i]
		if unescape && c == '\\' && i+1 < n {
			if next := s[i+1]; next == quote || next == '\\' {
				b.WriteByte(next)
				i += 2
				continue
			}
		}
		if c == quote {
			return b.String(), i + 1
		}
		b.WriteByte(c)
		i++
	}

	// 未闭合引号：返回已读内容
	return b.String(), i
}

// readBareWord 读取裸词，直到下一个未转义空白
func readBareWord(s string, start int) (string, int) {
	var b strings.Builder
	i, n := start, len(s)

	for i < n {
		c := s[i]
		if c == ' ' || c == '\t' {
			break
		}
		// 反斜杠转义空白，使其成为词的一部分
		if c == '\\' && i+1 < n {
			if next := s[i+1]; next == ' ' || next == '\t' {
				b.WriteByte(next)
				i += 2
				continue
			}
		}
		b.WriteByte(c)
		i++
	}

	return b.String(), i
}
