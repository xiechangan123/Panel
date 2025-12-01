package apache

import (
	"fmt"
	"sort"
	"strings"
)

// ExportOptions 定义导出选项
type ExportOptions struct {
	// IndentStyle 缩进样式：使用空格还是制表符
	IndentStyle string // "spaces" 或 "tabs"

	// IndentSize 缩进大小（仅当IndentStyle为"spaces"时有效）
	IndentSize int

	// SortDirectives 是否对指令进行排序
	SortDirectives bool

	// IncludeComments 是否包含注释
	IncludeComments bool

	// PreserveEmptyLines 是否保留空行
	PreserveEmptyLines bool

	// FormatStyle 格式化风格
	FormatStyle string // "compact", "standard", "verbose"
}

// DefaultExportOptions 返回默认的导出选项
func DefaultExportOptions() *ExportOptions {
	return &ExportOptions{
		IndentStyle:        "spaces",
		IndentSize:         4,
		SortDirectives:     false,
		IncludeComments:    true,
		PreserveEmptyLines: true,
		FormatStyle:        "standard",
	}
}

// Export 导出整个配置为Apache配置文件格式
func (c *Config) Export() string {
	return c.ExportWithOptions(DefaultExportOptions())
}

// ExportWithOptions 使用指定选项导出配置
func (c *Config) ExportWithOptions(options *ExportOptions) string {
	var builder strings.Builder
	var items []exportItem

	// 收集所有需要导出的项目
	// 添加全局注释
	if options.IncludeComments {
		for _, comment := range c.Comments {
			items = append(items, exportItem{
				line: comment.Line,
				item: comment,
				typ:  "comment",
			})
		}
	}

	// 添加全局指令
	for _, directive := range c.Directives {
		items = append(items, exportItem{
			line: directive.Line,
			item: directive,
			typ:  "directive",
		})
	}

	// 添加虚拟主机
	for _, vhost := range c.VirtualHosts {
		items = append(items, exportItem{
			line: vhost.Line,
			item: vhost,
			typ:  "virtualhost",
		})
	}

	// 如果不需要保持原始顺序，按行号排序
	if !options.SortDirectives {
		sort.Slice(items, func(i, j int) bool {
			return items[i].line < items[j].line
		})
	}

	// 导出所有项目
	for i, item := range items {
		switch item.typ {
		case "comment":
			comment := item.item.(*Comment)
			builder.WriteString(comment.ExportWithOptions(options, 0))
		case "directive":
			directive := item.item.(*Directive)
			builder.WriteString(directive.ExportWithOptions(options, 0))
		case "virtualhost":
			vhost := item.item.(*VirtualHost)
			builder.WriteString(vhost.ExportWithOptions(options, 0))
		}

		// 添加换行符
		if options.PreserveEmptyLines && i < len(items)-1 {
			// 检查是否需要添加空行
			nextItem := items[i+1]
			if shouldAddEmptyLine(item, nextItem, options) {
				builder.WriteString("\n")
			}
		}
	}

	return strings.TrimSpace(builder.String())
}

// ExportWithOptions 导出指令
func (d *Directive) ExportWithOptions(options *ExportOptions, indent int) string {
	var builder strings.Builder

	// 如果是块指令，使用Block的导出方法
	if d.Block != nil {
		return d.Block.ExportWithOptions(options, indent)
	}

	// 添加缩进
	builder.WriteString(getIndent(options, indent))

	// 指令名称
	builder.WriteString(d.Name)

	// 指令参数
	if len(d.Args) > 0 {
		builder.WriteString(" ")
		for i, arg := range d.Args {
			if i > 0 {
				builder.WriteString(" ")
			}

			// 如果参数包含空格，需要引用
			if strings.Contains(arg, " ") && !strings.HasPrefix(arg, "\"") {
				builder.WriteString(fmt.Sprintf(`"%s"`, arg))
			} else {
				builder.WriteString(arg)
			}
		}
	}

	builder.WriteString("\n")
	return builder.String()
}

// ExportWithOptions 导出虚拟主机
func (v *VirtualHost) ExportWithOptions(options *ExportOptions, indent int) string {
	var builder strings.Builder

	// 开始标签
	builder.WriteString(getIndent(options, indent))
	builder.WriteString("<")
	builder.WriteString(v.Name)
	if len(v.Args) > 0 {
		builder.WriteString(" ")
		builder.WriteString(strings.Join(v.Args, " "))
	}
	builder.WriteString(">\n")

	// 收集虚拟主机内的项目
	var items []exportItem

	// 添加注释
	if options.IncludeComments {
		for _, comment := range v.Comments {
			items = append(items, exportItem{
				line: comment.Line,
				item: comment,
				typ:  "comment",
			})
		}
	}

	// 添加指令
	for _, directive := range v.Directives {
		items = append(items, exportItem{
			line: directive.Line,
			item: directive,
			typ:  "directive",
		})
	}

	// 排序
	if !options.SortDirectives {
		sort.Slice(items, func(i, j int) bool {
			return items[i].line < items[j].line
		})
	}

	// 导出虚拟主机内容
	for _, item := range items {
		switch item.typ {
		case "comment":
			comment := item.item.(*Comment)
			builder.WriteString(comment.ExportWithOptions(options, indent+1))
		case "directive":
			directive := item.item.(*Directive)
			builder.WriteString(directive.ExportWithOptions(options, indent+1))
		}
	}

	// 结束标签
	builder.WriteString(getIndent(options, indent))
	builder.WriteString("</")
	builder.WriteString(v.Name)
	builder.WriteString(">\n")

	return builder.String()
}

// ExportWithOptions 导出注释
func (c *Comment) ExportWithOptions(options *ExportOptions, indent int) string {
	var builder strings.Builder

	// 添加缩进
	builder.WriteString(getIndent(options, indent))

	// 注释内容
	builder.WriteString("# ")
	builder.WriteString(c.Text)
	builder.WriteString("\n")

	return builder.String()
}

// ExportWithOptions 导出块指令
func (b *Block) ExportWithOptions(options *ExportOptions, indent int) string {
	var builder strings.Builder

	// 开始标签
	builder.WriteString(getIndent(options, indent))
	builder.WriteString("<")
	builder.WriteString(b.Type)
	if len(b.Args) > 0 {
		builder.WriteString(" ")
		for i, arg := range b.Args {
			if i > 0 {
				builder.WriteString(" ")
			}
			// 如果参数包含空格，需要引用
			if strings.Contains(arg, " ") && !strings.HasPrefix(arg, "\"") {
				builder.WriteString(fmt.Sprintf(`"%s"`, arg))
			} else {
				builder.WriteString(arg)
			}
		}
	}
	builder.WriteString(">\n")

	// 块内指令和注释
	allItems := make([]exportItem, 0, len(b.Directives)+len(b.Comments))

	// 添加指令
	for _, directive := range b.Directives {
		allItems = append(allItems, exportItem{item: directive, line: directive.Line, typ: "directive"})
	}

	// 添加注释
	if options.IncludeComments {
		for _, comment := range b.Comments {
			allItems = append(allItems, exportItem{item: comment, line: comment.Line, typ: "comment"})
		}
	}

	// 按行号排序
	if options.SortDirectives {
		sort.Slice(allItems, func(i, j int) bool {
			return allItems[i].line < allItems[j].line
		})
	}

	// 导出所有项目
	for i, item := range allItems {
		switch item.typ {
		case "comment":
			comment := item.item.(*Comment)
			builder.WriteString(comment.ExportWithOptions(options, indent+1))
		case "directive":
			directive := item.item.(*Directive)
			builder.WriteString(directive.ExportWithOptions(options, indent+1))
		}

		// 添加空行（如果需要）
		if options.PreserveEmptyLines && i < len(allItems)-1 {
			nextItem := allItems[i+1]
			if shouldAddEmptyLine(item, nextItem, options) {
				builder.WriteString("\n")
			}
		}
	}

	// 结束标签
	builder.WriteString(getIndent(options, indent))
	builder.WriteString("</")
	builder.WriteString(b.Type)
	builder.WriteString(">\n")

	return builder.String()
}

// exportItem 用于排序的导出项目
type exportItem struct {
	line int
	item interface{}
	typ  string
}

// getIndent 获取缩进字符串
func getIndent(options *ExportOptions, level int) string {
	if level <= 0 {
		return ""
	}

	if options.IndentStyle == "tabs" {
		return strings.Repeat("\t", level)
	}
	return strings.Repeat(" ", level*options.IndentSize)
}

// shouldAddEmptyLine 判断是否应该添加空行
func shouldAddEmptyLine(current, next exportItem, options *ExportOptions) bool {
	if !options.PreserveEmptyLines {
		return false
	}

	// 根据格式化风格决定
	switch options.FormatStyle {
	case "verbose":
		return true
	case "compact":
		return false
	case "standard":
		// 在不同类型之间添加空行（除了注释）
		if current.typ != next.typ && current.typ != "comment" && next.typ != "comment" {
			return true
		}
		return false
	}

	return false
}
