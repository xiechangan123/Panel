package nginx

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tufanbarisyildirim/gonginx/config"
	"github.com/tufanbarisyildirim/gonginx/dumper"
	"github.com/tufanbarisyildirim/gonginx/parser"
)

// Parser Nginx vhost 配置解析器
type Parser struct {
	cfg     *config.Config
	cfgPath string // 配置文件路径
}

func NewParser(website ...string) (*Parser, error) {
	str := DefaultConf
	cfgPath := ""
	if len(website) != 0 && website[0] != "" {
		cfgPath = fmt.Sprintf("/opt/ace/sites/%s/config/nginx.conf", website[0])
		if cfg, err := os.ReadFile(cfgPath); err == nil {
			str = string(cfg)
		} else {
			return nil, err
		}
	}

	p := parser.NewStringParser(str, parser.WithSkipIncludeParsingErr(), parser.WithSkipValidDirectivesErr())
	cfg, err := p.Parse()
	if err != nil {
		return nil, err
	}

	return &Parser{cfg: cfg, cfgPath: cfgPath}, nil
}

// NewParserFromFile 从指定文件路径创建解析器
func NewParserFromFile(filePath string) (*Parser, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	p := parser.NewStringParser(string(content), parser.WithSkipIncludeParsingErr(), parser.WithSkipValidDirectivesErr())
	cfg, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &Parser{cfg: cfg, cfgPath: filePath}, nil
}

func (p *Parser) Config() *config.Config {
	return p.cfg
}

// Find 查找指令，如: Find("server.listen")
func (p *Parser) Find(key string) ([]config.IDirective, error) {
	parts := strings.Split(key, ".")
	var block *config.Block
	var ok bool
	block = p.cfg.Block
	for i := 0; i < len(parts)-1; i++ {
		key = parts[i]
		directives := block.FindDirectives(key)
		if len(directives) == 0 {
			return nil, fmt.Errorf("given key %s not found", key)
		}
		if len(directives) > 1 {
			return nil, errors.New("multiple directives found")
		}
		block, ok = directives[0].GetBlock().(*config.Block)
		if !ok {
			return nil, errors.New("block is not *config.Block")
		}
	}

	var result []config.IDirective
	for _, dir := range block.GetDirectives() {
		if dir.GetName() == parts[len(parts)-1] {
			result = append(result, dir)
		}
	}

	return result, nil
}

// FindOne 查找单个指令，如: FindOne("server.server_name")
func (p *Parser) FindOne(key string) (config.IDirective, error) {
	directives, err := p.Find(key)
	if err != nil {
		return nil, err
	}
	if len(directives) == 0 {
		return nil, fmt.Errorf("given key %s not found", key)
	}
	if len(directives) > 1 {
		return nil, fmt.Errorf("multiple directives found for %s", key)
	}

	return directives[0], nil
}

// Clear 移除指令，如: Clear("server.server_name")
func (p *Parser) Clear(key string) error {
	parts := strings.Split(key, ".")
	last := parts[len(parts)-1]
	parts = parts[:len(parts)-1]

	var block *config.Block
	var ok bool
	block = p.cfg.Block
	for i := 0; i < len(parts); i++ {
		directives := block.FindDirectives(parts[i])
		if len(directives) == 0 {
			return fmt.Errorf("given key %s not found", parts[i])
		}
		if len(directives) > 1 {
			return fmt.Errorf("multiple directives found for %s", parts[i])
		}
		block, ok = directives[0].GetBlock().(*config.Block)
		if !ok {
			return errors.New("block is not *config.Block")
		}
	}

	var newDirectives []config.IDirective
	for _, directive := range block.GetDirectives() {
		if directive.GetName() != last {
			newDirectives = append(newDirectives, directive)
		}
	}
	block.Directives = newDirectives

	return nil
}

// Set 设置指令，如: Set("server.index", []Directive{...})
func (p *Parser) Set(key string, directives []*config.Directive, after ...string) error {
	parts := strings.Split(key, ".")

	var block *config.Block
	var blockDirective config.IDirective
	var ok bool
	block = p.cfg.Block
	for i := 0; i < len(parts); i++ {
		sub := block.FindDirectives(parts[i])
		if len(sub) == 0 {
			return fmt.Errorf("given key %s not found", parts[i])
		}
		if len(sub) > 1 {
			return fmt.Errorf("multiple directives found for %s", parts[i])
		}
		block, ok = sub[0].GetBlock().(*config.Block)
		if !ok {
			return errors.New("block is not *config.Block")
		}
		blockDirective = sub[0]
	}

	iDirectives := make([]config.IDirective, 0, len(directives))
	for _, directive := range directives {
		directive.SetParent(blockDirective)
		iDirectives = append(iDirectives, directive)
	}

	if len(after) == 0 {
		block.Directives = append(block.Directives, iDirectives...)
	} else {
		insertIndex := -1
		for i, d := range block.Directives {
			if d.GetName() == after[0] {
				insertIndex = i + 1
				break
			}
		}
		if insertIndex == -1 {
			return fmt.Errorf("after directive %s not found", after[0])
		}

		block.Directives = append(
			block.Directives[:insertIndex],
			append(iDirectives, block.Directives[insertIndex:]...)...,
		)
	}

	return nil
}

// Dump 将指令结构导出为配置内容
func (p *Parser) Dump() string {
	return dumper.DumpConfig(p.cfg, dumper.IndentedStyle)
}

// Save 保存配置到文件
func (p *Parser) Save() error {
	content := p.Dump()
	if err := os.WriteFile(p.cfgPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}

	return nil
}

// SetConfigPath 设置配置文件路径
func (p *Parser) SetConfigPath(path string) {
	p.cfgPath = path
}

func (p *Parser) slices2Parameters(slices []string) []config.Parameter {
	var parameters []config.Parameter
	for _, slice := range slices {
		parameters = append(parameters, config.Parameter{Value: slice})
	}
	return parameters
}

func (p *Parser) parameters2Slices(parameters []config.Parameter) []string {
	var s []string
	for _, parameter := range parameters {
		s = append(s, parameter.Value)
	}
	return s
}
