package apache

import (
	"io"
	"strings"
	"unicode"
)

// TokenType 表示 token 的类型
type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF
	NEWLINE
	COMMENT
	DIRECTIVE
	STRING
	LBRACE    // <
	RBRACE    // >
	SLASH     // /
	COLON     // :
	SEMICOLON // ;
	EQUAL     // =
	QUOTE     // "
	VIRTUALHOST
	BLOCKDIRECTIVE // Directory, Location 等块指令
)

// Token 表示一个词法单元
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// Lexer 词法分析器
type Lexer struct {
	current rune
	line    int
	column  int
	buf     []rune
	pos     int
	content string
}

// NewLexer 创建一个新的词法分析器
func NewLexer(input io.Reader) (*Lexer, error) {
	// 读取全部内容到字符串
	content := new(strings.Builder)
	_, err := io.Copy(content, input)
	if err != nil {
		return nil, err
	}

	l := &Lexer{
		line:    1,
		column:  0,
		content: content.String(),
		buf:     []rune(content.String()),
		pos:     -1,
	}

	l.readChar() // 初始化第一个字符
	return l, nil
}

// readChar 读取下一个字符
func (l *Lexer) readChar() {
	l.pos++
	if l.pos >= len(l.buf) {
		l.current = 0 // EOF
	} else {
		l.current = l.buf[l.pos]
	}

	l.column++
	if l.current == '\n' {
		l.line++
		l.column = 0
	}
}

// skipWhitespace 跳过空白字符
func (l *Lexer) skipWhitespace() {
	for l.current == ' ' || l.current == '\t' || l.current == '\r' {
		l.readChar()
	}
}

// readString 读取字符串字面量
func (l *Lexer) readString(delimiter rune) string {
	var result strings.Builder
	l.readChar() // 跳过开始的引号

	for l.current != delimiter && l.current != 0 {
		if l.current == '\\' {
			l.readChar()
			if l.current != 0 {
				// 保持转义字符的原始形式
				result.WriteRune('\\')
				result.WriteRune(l.current)
				l.readChar()
			}
		} else {
			result.WriteRune(l.current)
			l.readChar()
		}
	}

	return result.String()
}

// readIdentifier 读取标识符或指令名
func (l *Lexer) readIdentifier() string {
	var result strings.Builder

	for unicode.IsLetter(l.current) || unicode.IsDigit(l.current) || l.current == '_' || l.current == '-' || l.current == '.' || l.current == ':' || l.current == '/' || l.current == '$' || l.current == '@' || l.current == '%' || l.current == '{' || l.current == '}' || l.current == '?' || l.current == '&' || l.current == '=' || l.current == '+' {
		result.WriteRune(l.current)
		l.readChar()
	}

	return result.String()
}

// readWord 读取单词（可能包含特殊字符）
func (l *Lexer) readWord() string {
	var result strings.Builder

	for l.current != 0 && l.current != ' ' && l.current != '\t' && l.current != '\n' && l.current != '\r' &&
		l.current != '<' && l.current != '>' && l.current != '"' && l.current != '\'' {
		result.WriteRune(l.current)
		l.readChar()
	}

	return result.String()
}

// readComment 读取注释
func (l *Lexer) readComment() string {
	var result strings.Builder
	l.readChar() // 跳过 #

	// 跳过 # 后面的第一个空格（如果有的话）
	if l.current == ' ' {
		l.readChar()
	}

	for l.current != '\n' && l.current != 0 {
		result.WriteRune(l.current)
		l.readChar()
	}

	return result.String()
}

// isVirtualHostDirective 检查是否是虚拟主机指令
func (l *Lexer) isVirtualHostDirective(identifier string) bool {
	return strings.EqualFold(identifier, "VirtualHost")
}

// isBlockDirective 检查是否是块指令
func (l *Lexer) isBlockDirective(identifier string) bool {
	blockDirectives := []string{
		"Directory", "DirectoryMatch", "Location", "LocationMatch",
		"Files", "FilesMatch", "Limit", "LimitExcept", "RequireAll", "RequireAny", "RequireNone",
		"IfModule", "IfDefine", "IfVersion", "Proxy",
	}

	for _, blockDir := range blockDirectives {
		if strings.EqualFold(identifier, blockDir) {
			return true
		}
	}
	return false
}

// NextToken 获取下一个 token
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	switch l.current {
	case '#':
		tok.Type = COMMENT
		tok.Value = l.readComment()
	case '\n':
		tok.Type = NEWLINE
		tok.Value = "\n"
		l.readChar()
	case '<':
		// 检查是否是虚拟主机或目录块
		l.readChar() // 跳过 <

		// 检查是否是结束标签
		isClosing := false
		if l.current == '/' {
			isClosing = true
			l.readChar()
		}

		identifier := l.readIdentifier()

		// 如果无法读取到有效的标识符，这可能是无效语法
		if identifier == "" {
			// 将此作为ILLEGAL token处理
			tok.Type = ILLEGAL
			tok.Value = "<"
			return tok
		}

		// 跳过空白字符和参数
		l.skipWhitespace()
		var args []string
		for l.current != '>' && l.current != 0 {
			// 记录当前位置，防止无限循环
			oldPos := l.pos

			if l.current == '"' || l.current == '\'' {
				// 保留引号
				quoteChar := l.current
				arg := string(quoteChar) + l.readString(l.current) + string(quoteChar)
				args = append(args, arg)
				l.readChar() // 跳过结束引号
			} else {
				arg := l.readWord()
				if arg != "" {
					args = append(args, arg)
				}
			}
			l.skipWhitespace()

			// 如果位置没有前进，说明遇到了无法处理的字符，退出循环防止死循环
			if l.pos == oldPos {
				// 尝试跳过一个字符继续
				if l.current != 0 {
					l.readChar()
				}
				break
			}
		}

		if l.current == '>' {
			l.readChar() // 跳过 >
		}

		if l.isVirtualHostDirective(identifier) {
			tok.Type = VIRTUALHOST
			if isClosing {
				tok.Value = "/" + identifier
			} else {
				tok.Value = identifier
				if len(args) > 0 {
					tok.Value += " " + strings.Join(args, " ")
				}
			}
		} else if l.isBlockDirective(identifier) {
			// 识别为块指令
			tok.Type = BLOCKDIRECTIVE
			if isClosing {
				tok.Value = "/" + identifier
			} else {
				tok.Value = identifier
				if len(args) > 0 {
					tok.Value += " " + strings.Join(args, " ")
				}
			}
		} else {
			tok.Type = DIRECTIVE
			if isClosing {
				tok.Value = "/" + identifier
			} else {
				tok.Value = identifier
				if len(args) > 0 {
					tok.Value += " " + strings.Join(args, " ")
				}
			}
		}
	case '>':
		tok.Type = RBRACE
		tok.Value = ">"
		l.readChar()
	case '"':
		tok.Type = STRING
		// 保留引号
		tok.Value = `"` + l.readString('"') + `"`
		l.readChar()
	case '\'':
		tok.Type = STRING
		// 保留引号
		tok.Value = "'" + l.readString('\'') + "'"
		l.readChar()
	case 0:
		tok.Type = EOF
		tok.Value = ""
	default:
		if unicode.IsLetter(l.current) {
			identifier := l.readIdentifier()
			tok.Type = DIRECTIVE
			tok.Value = identifier
		} else {
			// 读取其他类型的单词
			word := l.readWord()
			if word != "" {
				tok.Type = STRING
				tok.Value = word
			} else {
				tok.Type = ILLEGAL
				tok.Value = string(l.current)
				l.readChar()
			}
		}
	}

	return tok
}

// PeekToken 预览下一个 token 而不移动位置
func (l *Lexer) PeekToken() Token {
	// 保存当前状态
	savedPos := l.pos
	savedLine := l.line
	savedColumn := l.column
	savedCurrent := l.current

	// 获取下一个 token
	token := l.NextToken()

	// 恢复状态
	l.pos = savedPos
	l.line = savedLine
	l.column = savedColumn
	l.current = savedCurrent

	return token
}
