package openlitespeed

import (
	"fmt"
	"os"
	"strings"
)

// OpenLiteSpeed 纯文本配置：`key value` 行、以 `{` 结尾的块、`}` 结束块、`key <<<SIGN` 多行值、`#` 注释

// Node 配置节点
type Node interface {
	render(b *strings.Builder, depth int)
}

// Comment 注释，不含 # 前缀
type Comment struct {
	Text string
}

// Directive 键值指令，Multiline 为 true 时 Value 为多行文本并以 heredoc 形式输出
type Directive struct {
	Name      string
	Value     string
	Multiline bool
}

// Block 块，Arg 为块名后的参数，如 context 的 uri、extprocessor 的名称
type Block struct {
	nodeList
	Name string
	Arg  string
}

// Config 配置文件根
type Config struct {
	nodeList
}

// nodeList 节点列表，Config 与 Block 共享查询与修改方法
type nodeList struct {
	Nodes []Node
}

const indentUnit = "  "

// Parse 解析配置文本
func Parse(content string) (*Config, error) {
	cfg := &Config{}
	stack := []*nodeList{&cfg.nodeList}
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		// 行尾反斜杠续行
		for strings.HasSuffix(line, `\`) && i+1 < len(lines) {
			i++
			line = strings.TrimSuffix(line, `\`) + strings.TrimSpace(lines[i])
		}
		if line == "" {
			continue
		}
		cur := stack[len(stack)-1]

		if strings.HasPrefix(line, "#") {
			cur.Nodes = append(cur.Nodes, &Comment{Text: strings.TrimPrefix(line, "#")})
			continue
		}
		if strings.HasPrefix(line, "}") {
			if len(stack) == 1 {
				return nil, fmt.Errorf("line %d: unexpected '}'", i+1)
			}
			stack = stack[:len(stack)-1]
			// } 之后可能紧跟新内容
			if rest := strings.TrimSpace(line[1:]); rest != "" {
				lines[i] = rest
				i--
			}
			continue
		}
		if strings.HasSuffix(line, "{") {
			name, value := splitKeyValue(strings.TrimSpace(strings.TrimSuffix(line, "{")))
			if name == "" {
				return nil, fmt.Errorf("line %d: block without name", i+1)
			}
			block := &Block{Name: name, Arg: value}
			cur.Nodes = append(cur.Nodes, block)
			stack = append(stack, &block.nodeList)
			continue
		}

		name, value := splitKeyValue(line)
		if sign, ok := strings.CutPrefix(value, "<<<"); ok {
			sign = strings.TrimSpace(sign)
			var body []string
			closed := false
			for i+1 < len(lines) {
				i++
				if strings.TrimSpace(lines[i]) == sign {
					closed = true
					break
				}
				body = append(body, lines[i])
			}
			if !closed {
				return nil, fmt.Errorf("unterminated multiline value for %s", name)
			}
			cur.Nodes = append(cur.Nodes, &Directive{Name: name, Value: strings.Join(body, "\n"), Multiline: true})
			continue
		}
		cur.Nodes = append(cur.Nodes, &Directive{Name: name, Value: value})
	}

	if len(stack) != 1 {
		return nil, fmt.Errorf("unclosed block")
	}

	return cfg, nil
}

// ParseFile 解析配置文件
func ParseFile(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(content))
}

// splitKeyValue 按首个空白拆分键与值
func splitKeyValue(line string) (string, string) {
	idx := strings.IndexAny(line, " \t")
	if idx < 0 {
		return line, ""
	}
	return line[:idx], strings.TrimSpace(line[idx+1:])
}

// String 渲染为配置文本
func (c *Config) String() string {
	var b strings.Builder
	for i, n := range c.Nodes {
		// 顶层块之间空一行
		if _, ok := n.(*Block); ok && i > 0 {
			b.WriteByte('\n')
		}
		n.render(&b, 0)
	}
	return b.String()
}

func (c *Comment) render(b *strings.Builder, depth int) {
	b.WriteString(strings.Repeat(indentUnit, depth))
	b.WriteByte('#')
	b.WriteString(c.Text)
	b.WriteByte('\n')
}

func (d *Directive) render(b *strings.Builder, depth int) {
	indent := strings.Repeat(indentUnit, depth)
	if d.Multiline {
		sign := "END_" + d.Name
		_, _ = fmt.Fprintf(b, "%s%s <<<%s\n%s\n%s\n", indent, d.Name, sign, d.Value, sign)
		return
	}
	if d.Value == "" {
		_, _ = fmt.Fprintf(b, "%s%s\n", indent, d.Name)
		return
	}
	_, _ = fmt.Fprintf(b, "%s%-24s %s\n", indent, d.Name, d.Value)
}

func (blk *Block) render(b *strings.Builder, depth int) {
	indent := strings.Repeat(indentUnit, depth)
	b.WriteString(indent)
	b.WriteString(blk.Name)
	if blk.Arg != "" {
		b.WriteByte(' ')
		b.WriteString(blk.Arg)
	}
	b.WriteString(" {\n")
	for _, n := range blk.Nodes {
		n.render(b, depth+1)
	}
	b.WriteString(indent)
	b.WriteString("}\n")
}

// ========== 查询与修改，名称比较大小写不敏感 ==========

// Append 追加节点
func (l *nodeList) Append(nodes ...Node) {
	l.Nodes = append(l.Nodes, nodes...)
}

// Directive 返回首个匹配名称的指令
func (l *nodeList) Directive(name string) *Directive {
	for _, n := range l.Nodes {
		if d, ok := n.(*Directive); ok && strings.EqualFold(d.Name, name) {
			return d
		}
	}
	return nil
}

// Directives 返回所有匹配名称的指令
func (l *nodeList) Directives(name string) []*Directive {
	var out []*Directive
	for _, n := range l.Nodes {
		if d, ok := n.(*Directive); ok && strings.EqualFold(d.Name, name) {
			out = append(out, d)
		}
	}
	return out
}

// Value 返回指令值，不存在返回空串
func (l *nodeList) Value(name string) string {
	if d := l.Directive(name); d != nil {
		return d.Value
	}
	return ""
}

// Add 追加一条指令
func (l *nodeList) Add(name, value string) *Directive {
	d := &Directive{Name: name, Value: value}
	l.Nodes = append(l.Nodes, d)
	return d
}

// Set 设置指令：存在则更新，否则追加
func (l *nodeList) Set(name, value string) *Directive {
	if d := l.Directive(name); d != nil {
		d.Value = value
		return d
	}
	return l.Add(name, value)
}

// Remove 删除所有匹配名称的指令
func (l *nodeList) Remove(name string) {
	l.RemoveFunc(func(n Node) bool {
		d, ok := n.(*Directive)
		return ok && strings.EqualFold(d.Name, name)
	})
}

// RemoveFunc 删除满足谓词的节点
func (l *nodeList) RemoveFunc(pred func(Node) bool) {
	kept := l.Nodes[:0]
	for _, n := range l.Nodes {
		if !pred(n) {
			kept = append(kept, n)
		}
	}
	l.Nodes = kept
}

// Block 返回首个匹配名称（及可选参数）的块
func (l *nodeList) Block(name string, arg ...string) *Block {
	for _, n := range l.Nodes {
		if b, ok := n.(*Block); ok && strings.EqualFold(b.Name, name) {
			if len(arg) == 0 || b.Arg == arg[0] {
				return b
			}
		}
	}
	return nil
}

// Blocks 返回所有匹配名称的块
func (l *nodeList) Blocks(name string) []*Block {
	var out []*Block
	for _, n := range l.Nodes {
		if b, ok := n.(*Block); ok && strings.EqualFold(b.Name, name) {
			out = append(out, b)
		}
	}
	return out
}

// AddBlock 追加一个块
func (l *nodeList) AddBlock(name, arg string) *Block {
	b := &Block{Name: name, Arg: arg}
	l.Nodes = append(l.Nodes, b)
	return b
}

// Comments 返回所有注释文本
func (l *nodeList) Comments() []string {
	var out []string
	for _, n := range l.Nodes {
		if c, ok := n.(*Comment); ok {
			out = append(out, strings.TrimSpace(c.Text))
		}
	}
	return out
}

// Meta 读取形如 `# ace:key value` 的元数据注释
func (l *nodeList) Meta(key string) string {
	for _, c := range l.Comments() {
		if rest, ok := strings.CutPrefix(c, "ace:"+key+" "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// AddMeta 写入元数据注释
func (l *nodeList) AddMeta(key, value string) {
	l.Nodes = append(l.Nodes, &Comment{Text: " ace:" + key + " " + value})
}
