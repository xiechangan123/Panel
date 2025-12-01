package apache

import (
	"strings"
)

// Config Apache 配置文件的 AST 根节点
type Config struct {
	Directives   []*Directive   `json:"directives"`
	VirtualHosts []*VirtualHost `json:"virtual_hosts"`
	Comments     []*Comment     `json:"comments"`
	Includes     []*Include     `json:"includes"`
}

// Directive Apache 指令
type Directive struct {
	Name   string   `json:"name"`
	Args   []string `json:"args"`
	Line   int      `json:"line"`
	Column int      `json:"column"`
	Block  *Block   `json:"block,omitempty"` // 对于有块的指令如 <Directory>
}

// VirtualHost 虚拟主机配置
type VirtualHost struct {
	Name       string       `json:"name"`
	Args       []string     `json:"args"` // 通常是 IP:Port
	Line       int          `json:"line"`
	Column     int          `json:"column"`
	Directives []*Directive `json:"directives"`
	Comments   []*Comment   `json:"comments"` // 虚拟主机内的注释
}

// Block 配置块，如 <Directory>, <Location> 等
type Block struct {
	Type       string       `json:"type"` // Directory, Location, Files 等
	Args       []string     `json:"args"` // 块的参数
	Directives []*Directive `json:"directives"`
	Comments   []*Comment   `json:"comments"` // 块内注释
	Line       int          `json:"line"`
	Column     int          `json:"column"`
}

// Comment 注释
type Comment struct {
	Text   string `json:"text"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// Include 包含其他配置文件的指令
type Include struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// GetDirective 根据名称查找指令
func (c *Config) GetDirective(name string) *Directive {
	for _, dir := range c.Directives {
		if strings.EqualFold(dir.Name, name) {
			return dir
		}
	}
	return nil
}

// GetDirectives 根据名称查找所有匹配的指令
func (c *Config) GetDirectives(name string) []*Directive {
	var result []*Directive
	for _, dir := range c.Directives {
		if strings.EqualFold(dir.Name, name) {
			result = append(result, dir)
		}
	}
	return result
}

// GetVirtualHost 根据参数查找虚拟主机
func (c *Config) GetVirtualHost(args ...string) *VirtualHost {
	for _, vhost := range c.VirtualHosts {
		if len(vhost.Args) == len(args) {
			match := true
			for i, arg := range args {
				if vhost.Args[i] != arg {
					match = false
					break
				}
			}
			if match {
				return vhost
			}
		}
	}
	return nil
}

// AddVirtualHost 添加新虚拟主机到配置
func (c *Config) AddVirtualHost(args ...string) *VirtualHost {
	vhost := &VirtualHost{
		Name:       "VirtualHost",
		Args:       args,
		Directives: make([]*Directive, 0),
	}
	c.VirtualHosts = append(c.VirtualHosts, vhost)
	return vhost
}

// AddDirective 为虚拟主机添加指令
func (v *VirtualHost) AddDirective(name string, args ...string) *Directive {
	directive := &Directive{
		Name: name,
		Args: args,
	}
	v.Directives = append(v.Directives, directive)
	return directive
}

// GetDirective 在虚拟主机中根据名称查找指令
func (v *VirtualHost) GetDirective(name string) *Directive {
	for _, dir := range v.Directives {
		if strings.EqualFold(dir.Name, name) {
			return dir
		}
	}
	return nil
}

// GetDirectives 在虚拟主机中根据名称查找所有匹配的指令
func (v *VirtualHost) GetDirectives(name string) []*Directive {
	var result []*Directive
	for _, dir := range v.Directives {
		if strings.EqualFold(dir.Name, name) {
			result = append(result, dir)
		}
	}
	return result
}

// SetDirective 设置指令（如果存在则更新，不存在则添加）
func (v *VirtualHost) SetDirective(name string, args ...string) *Directive {
	// 查找现有指令
	for _, dir := range v.Directives {
		if strings.EqualFold(dir.Name, name) {
			dir.Args = args
			return dir
		}
	}
	// 不存在，添加新指令
	return v.AddDirective(name, args...)
}

// RemoveDirective 删除指令
func (v *VirtualHost) RemoveDirective(name string) bool {
	for i, dir := range v.Directives {
		if strings.EqualFold(dir.Name, name) {
			v.Directives = append(v.Directives[:i], v.Directives[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveDirectives 删除所有匹配名称的指令
func (v *VirtualHost) RemoveDirectives(name string) int {
	count := 0
	newDirectives := make([]*Directive, 0, len(v.Directives))
	for _, dir := range v.Directives {
		if strings.EqualFold(dir.Name, name) {
			count++
		} else {
			newDirectives = append(newDirectives, dir)
		}
	}
	v.Directives = newDirectives
	return count
}

// HasDirective 检查是否存在指定指令
func (v *VirtualHost) HasDirective(name string) bool {
	return v.GetDirective(name) != nil
}

// GetDirectiveValue 获取指令的第一个参数值
func (v *VirtualHost) GetDirectiveValue(name string) string {
	dir := v.GetDirective(name)
	if dir != nil && len(dir.Args) > 0 {
		return dir.Args[0]
	}
	return ""
}

// GetDirectiveValues 获取指令的所有参数值
func (v *VirtualHost) GetDirectiveValues(name string) []string {
	dir := v.GetDirective(name)
	if dir != nil {
		return dir.Args
	}
	return nil
}

// AddBlock 添加块指令（如 Directory, Location 等）
func (v *VirtualHost) AddBlock(blockType string, args ...string) *Directive {
	block := &Block{
		Type:       blockType,
		Args:       args,
		Directives: make([]*Directive, 0),
		Comments:   make([]*Comment, 0),
	}
	directive := &Directive{
		Name:  blockType,
		Args:  args,
		Block: block,
	}
	v.Directives = append(v.Directives, directive)
	return directive
}

// GetBlock 获取块指令
func (v *VirtualHost) GetBlock(blockType string, args ...string) *Block {
	for _, dir := range v.Directives {
		if dir.Block != nil && strings.EqualFold(dir.Block.Type, blockType) {
			// 如果指定了参数，需要匹配
			if len(args) > 0 {
				if len(dir.Block.Args) != len(args) {
					continue
				}
				match := true
				for i, arg := range args {
					if dir.Block.Args[i] != arg {
						match = false
						break
					}
				}
				if !match {
					continue
				}
			}
			return dir.Block
		}
	}
	return nil
}
