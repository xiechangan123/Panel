package apache

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Parser Apache 配置文件解析器
type Parser struct {
	lexer          *Lexer
	options        *ParseOptions
	includeStack   []string // 用于检测循环包含
	currentDepth   int      // 当前包含深度
	currentBaseDir string   // 当前文件的基础目录
}

// NewParser 创建一个新的 Apache 配置解析器（使用默认选项）
func NewParser(input io.Reader) (*Parser, error) {
	return NewParserWithOptions(input, DefaultParseOptions())
}

// NewParserWithOptions 创建一个带选项的 Apache 配置解析器
func NewParserWithOptions(input io.Reader, options *ParseOptions) (*Parser, error) {
	lexer, err := NewLexer(input)
	if err != nil {
		return nil, err
	}

	return &Parser{
		lexer:          lexer,
		options:        options,
		includeStack:   make([]string, 0),
		currentDepth:   0,
		currentBaseDir: options.BaseDir,
	}, nil
}

// Parse 解析 Apache 配置文件并返回 AST
func (p *Parser) Parse() (*Config, error) {
	config := &Config{
		Directives:   make([]*Directive, 0),
		VirtualHosts: make([]*VirtualHost, 0),
		Comments:     make([]*Comment, 0),
	}

	for {
		token := p.lexer.NextToken()
		if token.Type == EOF {
			break
		}

		switch token.Type {
		case COMMENT:
			comment := &Comment{
				Text:   token.Value,
				Line:   token.Line,
				Column: token.Column,
			}
			config.Comments = append(config.Comments, comment)

		case DIRECTIVE:
			directive, err := p.parseDirective(token)
			if err != nil {
				return nil, fmt.Errorf("error parsing directive: %w", err)
			}

			// 检查是否是Include类指令
			if p.options.ProcessIncludes && (strings.EqualFold(directive.Name, "Include") || strings.EqualFold(directive.Name, "IncludeOptional")) {
				includeConfig, err := p.processInclude(directive)
				if err != nil {
					// 对于IncludeOptional，如果文件不存在则忽略错误
					if strings.EqualFold(directive.Name, "IncludeOptional") && os.IsNotExist(err) {
						// 仍然记录Include指令，但不合并内容
						include := &Include{
							Path:   directive.Args[0],
							Line:   directive.Line,
							Column: directive.Column,
						}
						config.Includes = append(config.Includes, include)
						continue
					}
					return nil, fmt.Errorf("error processing include file '%s': %w", directive.Args[0], err)
				}

				if includeConfig != nil {
					// 合并包含的配置到当前配置
					config.Directives = append(config.Directives, includeConfig.Directives...)
					config.VirtualHosts = append(config.VirtualHosts, includeConfig.VirtualHosts...)
					config.Comments = append(config.Comments, includeConfig.Comments...)
					config.Includes = append(config.Includes, includeConfig.Includes...)
				}

				// 记录Include指令
				include := &Include{
					Path:   directive.Args[0],
					Line:   directive.Line,
					Column: directive.Column,
				}
				config.Includes = append(config.Includes, include)
			} else {
				config.Directives = append(config.Directives, directive)
			}

		case VIRTUALHOST:
			vhost, err := p.parseVirtualHost(token)
			if err != nil {
				return nil, fmt.Errorf("error parsing virtual host: %w", err)
			}
			config.VirtualHosts = append(config.VirtualHosts, vhost)

		case BLOCKDIRECTIVE:
			block, err := p.parseBlockDirective(token)
			if err != nil {
				return nil, fmt.Errorf("error parsing block directive: %w", err)
			}
			// 将块指令作为带Block的Directive添加到配置中
			directive := &Directive{
				Name:   block.Type,
				Args:   block.Args,
				Line:   block.Line,
				Column: block.Column,
				Block:  block,
			}
			config.Directives = append(config.Directives, directive)

		case NEWLINE:
			// 跳过换行符
			continue

		case ILLEGAL:
			return nil, fmt.Errorf("invalid syntax: '%s' at line %d, column %d", token.Value, token.Line, token.Column)

		default:
			return nil, fmt.Errorf("unknown token type: %v at line %d, column %d", token.Type, token.Line, token.Column)
		}
	}

	return config, nil
}

// parseDirective 解析单个指令
func (p *Parser) parseDirective(token Token) (*Directive, error) {
	directive := &Directive{
		Name:   token.Value,
		Line:   token.Line,
		Column: token.Column,
		Args:   make([]string, 0),
	}

	// 读取指令参数
	for {
		nextToken := p.lexer.PeekToken()
		if nextToken.Type == NEWLINE || nextToken.Type == EOF {
			break
		}

		argToken := p.lexer.NextToken()
		if argToken.Type == STRING || argToken.Type == DIRECTIVE {
			directive.Args = append(directive.Args, argToken.Value)
		}
	}

	return directive, nil
}

// parseVirtualHost 解析虚拟主机配置
func (p *Parser) parseVirtualHost(token Token) (*VirtualHost, error) {
	// 从 token.Value 中提取虚拟主机名称和参数
	parts := strings.Fields(token.Value)
	vhost := &VirtualHost{
		Name:       "VirtualHost",
		Line:       token.Line,
		Column:     token.Column,
		Directives: make([]*Directive, 0),
		Comments:   make([]*Comment, 0),
	}

	// 如果有参数，添加到 Args 中
	if len(parts) > 1 {
		vhost.Args = parts[1:]
	}

	// 跳过换行符到虚拟主机内容
	for {
		nextToken := p.lexer.NextToken()
		if nextToken.Type == EOF {
			return nil, fmt.Errorf("unexpected end of virtual host")
		}
		// 检查结束标签（可能是VIRTUALHOST或DIRECTIVE类型）
		if (nextToken.Type == VIRTUALHOST && strings.HasPrefix(nextToken.Value, "/VirtualHost")) ||
			(nextToken.Type == DIRECTIVE && strings.HasPrefix(nextToken.Value, "/VirtualHost")) {
			// 遇到结束标签
			break
		}
		if nextToken.Type == NEWLINE {
			continue
		}

		if nextToken.Type == DIRECTIVE {
			directive, err := p.parseDirective(nextToken)
			if err != nil {
				return nil, err
			}

			// 检查是否是Include类指令
			if p.options.ProcessIncludes && (strings.EqualFold(directive.Name, "Include") || strings.EqualFold(directive.Name, "IncludeOptional")) {
				includeConfig, err := p.processInclude(directive)
				if err != nil {
					// 对于IncludeOptional，如果文件不存在则忽略错误
					if strings.EqualFold(directive.Name, "IncludeOptional") && os.IsNotExist(err) {
						continue
					}
					return nil, fmt.Errorf("error processing include file '%s': %w", directive.Args[0], err)
				}

				if includeConfig != nil {
					// 合并包含的配置到虚拟主机
					vhost.Directives = append(vhost.Directives, includeConfig.Directives...)
					vhost.Comments = append(vhost.Comments, includeConfig.Comments...)
					// 注意：虚拟主机内的Include不应该包含其他虚拟主机，但如果包含了也要处理
					if len(includeConfig.VirtualHosts) > 0 {
						return nil, fmt.Errorf("include files inside virtual host cannot contain other virtual hosts")
					}
				}
			} else {
				vhost.Directives = append(vhost.Directives, directive)
			}
		} else if nextToken.Type == COMMENT {
			// 收集虚拟主机内的注释
			comment := &Comment{
				Text:   nextToken.Value,
				Line:   nextToken.Line,
				Column: nextToken.Column,
			}
			vhost.Comments = append(vhost.Comments, comment)
		}
	}

	return vhost, nil
}

// ParseFile 从文件解析 Apache 配置
func ParseFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func(file *os.File) { _ = file.Close() }(file)

	parser, err := NewParser(file)
	if err != nil {
		return nil, err
	}

	return parser.Parse()
}

// ParseString 从字符串解析 Apache 配置
func ParseString(content string) (*Config, error) {
	reader := strings.NewReader(content)
	parser, err := NewParser(reader)
	if err != nil {
		return nil, err
	}

	return parser.Parse()
}

// ParseStringWithOptions 从字符串解析 Apache 配置（带选项）
func ParseStringWithOptions(content string, options *ParseOptions) (*Config, error) {
	reader := strings.NewReader(content)
	parser, err := NewParserWithOptions(reader, options)
	if err != nil {
		return nil, err
	}

	return parser.Parse()
}

// ParseFileWithOptions 从文件解析 Apache 配置（带选项）
func ParseFileWithOptions(filename string, options *ParseOptions) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func(file *os.File) { _ = file.Close() }(file)

	// 设置基础目录为文件所在目录
	if options.BaseDir == "" {
		options.BaseDir = filepath.Dir(filename)
	}

	parser, err := NewParserWithOptions(file, options)
	if err != nil {
		return nil, err
	}

	// 设置当前文件路径用于循环检测
	absPath, _ := filepath.Abs(filename)
	parser.includeStack = append(parser.includeStack, absPath)

	return parser.Parse()
}

// processInclude 处理Include指令
func (p *Parser) processInclude(directive *Directive) (*Config, error) {
	if len(directive.Args) == 0 {
		return nil, fmt.Errorf("include directive missing file path argument")
	}

	// 检查递归深度
	if p.currentDepth >= p.options.MaxIncludeDepth {
		return nil, fmt.Errorf("include nesting depth exceeds limit %d", p.options.MaxIncludeDepth)
	}

	includePath := directive.Args[0]

	// 解析包含文件的完整路径
	var fullPath string
	var err error

	if filepath.IsAbs(includePath) {
		fullPath = includePath
	} else {
		// 相对路径基于当前文件所在目录
		fullPath = filepath.Join(p.currentBaseDir, includePath)
	}

	// 获取绝对路径用于循环检测
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of file '%s': %w", fullPath, err)
	}

	// 检查循环包含
	for _, stackPath := range p.includeStack {
		if stackPath == absPath {
			return nil, fmt.Errorf("circular include detected: %s", absPath)
		}
	}

	// 检查文件是否存在
	if _, err := os.Stat(fullPath); err != nil {
		return nil, err
	}

	// 打开并解析包含的文件
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open include file: %w", err)
	}
	defer func(file *os.File) { _ = file.Close() }(file)

	// 创建新的解析器选项，继承当前选项但更新基础目录
	includeOptions := *p.options
	includeOptions.BaseDir = filepath.Dir(fullPath)

	// 创建新的解析器实例
	includeParser, err := NewParserWithOptions(file, &includeOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create include file parser: %w", err)
	}

	// 设置包含解析器的状态
	includeParser.includeStack = append(p.includeStack, absPath)
	includeParser.currentDepth = p.currentDepth + 1
	includeParser.currentBaseDir = filepath.Dir(fullPath)

	// 解析包含的文件
	return includeParser.Parse()
}

// parseBlockDirective 解析块指令（如Directory, Location等）
func (p *Parser) parseBlockDirective(token Token) (*Block, error) {
	// 从token.Value中提取块类型和参数
	parts := strings.Fields(token.Value)
	block := &Block{
		Type:       parts[0],
		Line:       token.Line,
		Column:     token.Column,
		Directives: make([]*Directive, 0),
		Comments:   make([]*Comment, 0),
	}

	// 如果有参数，添加到Args中
	if len(parts) > 1 {
		block.Args = parts[1:]
	}

	// 跳过换行符到块内容
	for {
		nextToken := p.lexer.NextToken()
		if nextToken.Type == EOF {
			return nil, fmt.Errorf("unexpected end of block directive")
		}

		// 检查结束标签
		if (nextToken.Type == BLOCKDIRECTIVE && strings.HasPrefix(nextToken.Value, "/"+block.Type)) ||
			(nextToken.Type == DIRECTIVE && strings.HasPrefix(nextToken.Value, "/"+block.Type)) {
			// 遇到结束标签
			break
		}

		switch nextToken.Type {
		case NEWLINE:
			continue
		case DIRECTIVE:
			directive, err := p.parseDirective(nextToken)
			if err != nil {
				return nil, err
			}
			block.Directives = append(block.Directives, directive)
		case COMMENT:
			// 处理块内注释
			comment := &Comment{
				Text:   nextToken.Value,
				Line:   nextToken.Line,
				Column: nextToken.Column,
			}
			block.Comments = append(block.Comments, comment)
		default:
			continue
		}
	}

	return block, nil
}
