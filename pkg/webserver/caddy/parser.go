package caddy

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Caddyfile 文本模型：每行一条指令，首个 token 为名称、其余为参数；行尾单独的 `{` 开块，`}` 独占一行收块；
// 顶层带块的行是站点块（token 为逗号分隔的地址）、片段 `(name)` 或无 token 的全局选项块；
// `#` 开头的 token 至行尾为注释；参数支持双引号、反引号与 heredoc，行尾反斜杠续行

// Node 配置节点
type Node interface {
	render(b *strings.Builder, depth int)
}

// Comment 注释，不含 # 前缀
type Comment struct {
	Text string
}

// Blank 空行，仅用于生成时分组
type Blank struct{}

// Directive 指令行，Block 为 true 时带子块；Site 为 true 时是站点块，Tokens 为地址列表
type Directive struct {
	nodeList
	Tokens   []string
	Block    bool
	Site     bool
	Trailing string // 行尾注释
}

// Config 配置文件根
type Config struct {
	nodeList
}

// nodeList 节点列表，Config 与 Directive 共享查询与修改方法
type nodeList struct {
	Nodes []Node
}

type token struct {
	text    string
	line    int
	quoted  bool // 引号、反引号或 heredoc 包裹，不参与块结构判断
	comment bool
}

var heredocMarker = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Parse 解析配置文本
func Parse(content string) (*Config, error) {
	tokens, err := lex(content)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	stack := []*nodeList{&cfg.nodeList}
	for i := 0; i < len(tokens); {
		var toks []token
		trailing := ""
		line := tokens[i].line
		// 顶层地址列表以逗号结尾时跨行继续
		for i < len(tokens) {
			line = tokens[i].line
			for ; i < len(tokens) && tokens[i].line == line; i++ {
				if tokens[i].comment {
					trailing = tokens[i].text
					continue
				}
				toks = append(toks, tokens[i])
			}
			if len(stack) > 1 || len(toks) == 0 || !strings.HasSuffix(toks[len(toks)-1].text, ",") {
				break
			}
		}
		cur := stack[len(stack)-1]

		// 没有指令 token 的行只可能是纯注释行
		if len(toks) == 0 {
			cur.Nodes = append(cur.Nodes, &Comment{Text: trailing})
			continue
		}
		if !toks[0].quoted && toks[0].text == "}" {
			if len(toks) > 1 {
				return nil, fmt.Errorf("line %d: unexpected token after '}'", line)
			}
			if len(stack) == 1 {
				return nil, fmt.Errorf("line %d: unexpected '}'", line)
			}
			stack = stack[:len(stack)-1]
			if trailing != "" {
				stack[len(stack)-1].Nodes = append(stack[len(stack)-1].Nodes, &Comment{Text: trailing})
			}
			continue
		}

		block := false
		if last := toks[len(toks)-1]; !last.quoted && last.text == "{" {
			block = true
			toks = toks[:len(toks)-1]
		}
		d := &Directive{Block: block, Trailing: trailing}
		for _, t := range toks {
			d.Tokens = append(d.Tokens, t.text)
		}
		// 顶层带块且不是片段定义的行是站点块，地址以逗号分隔
		if len(stack) == 1 && block && len(d.Tokens) > 0 && !strings.HasPrefix(d.Tokens[0], "(") {
			d.Site = true
			d.Tokens = splitAddresses(d.Tokens)
		}
		cur.Nodes = append(cur.Nodes, d)
		if block {
			stack = append(stack, &d.nodeList)
		}
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

// lex 切分 token，逻辑行号用于分组，续行、引号与 heredoc 内的换行不增加行号
func lex(content string) ([]token, error) {
	s := strings.ReplaceAll(content, "\r\n", "\n")
	var tokens []token
	line := 1
	for i := 0; i < len(s); {
		ch := s[i]
		switch {
		case ch == '\n':
			line++
			i++
		case ch == ' ' || ch == '\t' || ch == '\r':
			i++
		case escapedNewline(s, i):
			i += 2
		case ch == '#':
			end := strings.IndexByte(s[i:], '\n')
			if end < 0 {
				end = len(s) - i
			}
			tokens = append(tokens, token{text: s[i+1 : i+end], line: line, comment: true})
			i += end
		case ch == '"':
			var val strings.Builder
			j := i + 1
			for ; j < len(s) && s[j] != '"'; j++ {
				if s[j] == '\\' && j+1 < len(s) && s[j+1] == '"' {
					j++
				}
				val.WriteByte(s[j])
			}
			if j >= len(s) {
				return nil, fmt.Errorf("line %d: unterminated quoted string", line)
			}
			tokens = append(tokens, token{text: val.String(), line: line, quoted: true})
			i = j + 1
		case ch == '`':
			end := strings.IndexByte(s[i+1:], '`')
			if end < 0 {
				return nil, fmt.Errorf("line %d: unterminated backtick string", line)
			}
			tokens = append(tokens, token{text: s[i+1 : i+1+end], line: line, quoted: true})
			i += end + 2
		case ch == '<' && strings.HasPrefix(s[i:], "<<") && isHeredoc(s[i+2:]):
			nl := strings.IndexByte(s[i:], '\n')
			marker := s[i+2 : i+nl]
			body, next, err := readHeredoc(s[i+nl+1:], marker)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
			tokens = append(tokens, token{text: body, line: line, quoted: true})
			i += nl + 1 + next
		default:
			j := i
			for j < len(s) && !strings.ContainsRune(" \t\r\n", rune(s[j])) && !escapedNewline(s, j) {
				j++
			}
			tokens = append(tokens, token{text: s[i:j], line: line})
			i = j
		}
	}
	return tokens, nil
}

// escapedNewline 位置 i 是否为反斜杠续行
func escapedNewline(s string, i int) bool {
	return s[i] == '\\' && i+1 < len(s) && s[i+1] == '\n'
}

// isHeredoc 判断 `<<` 之后是否为合法的 heredoc 标记且标记后紧跟换行
func isHeredoc(rest string) bool {
	nl := strings.IndexByte(rest, '\n')
	return nl > 0 && heredocMarker.MatchString(rest[:nl])
}

// readHeredoc 读取 heredoc 正文直到独立的结束标记行，按结束标记的缩进去除每行前导空白，返回正文与消费的字节数
func readHeredoc(s, marker string) (string, int, error) {
	consumed := 0
	var lines []string
	for consumed <= len(s) {
		end := strings.IndexByte(s[consumed:], '\n')
		physical := ""
		if end < 0 {
			physical = s[consumed:]
		} else {
			physical = s[consumed : consumed+end]
		}
		if trimmed := strings.TrimLeft(physical, " \t"); strings.HasPrefix(trimmed, marker) &&
			(len(trimmed) == len(marker) || strings.ContainsRune(" \t\r", rune(trimmed[len(marker)]))) {
			indent := physical[:len(physical)-len(trimmed)]
			for i, l := range lines {
				lines[i] = strings.TrimPrefix(l, indent)
			}
			return strings.Join(lines, "\n"), consumed + len(indent) + len(marker), nil
		}
		if end < 0 {
			break
		}
		lines = append(lines, physical)
		consumed += end + 1
	}
	return "", 0, fmt.Errorf("unterminated heredoc %s", marker)
}

// splitAddresses 去掉站点地址间的逗号
func splitAddresses(tokens []string) []string {
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		for part := range strings.SplitSeq(t, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

// ========== 渲染 ==========

// String 渲染为配置文本，顶层块之间空一行
func (c *Config) String() string {
	var b strings.Builder
	for i, n := range c.Nodes {
		if d, ok := n.(*Directive); ok && d.Block && i > 0 {
			if _, blank := c.Nodes[i-1].(*Blank); !blank {
				b.WriteByte('\n')
			}
		}
		n.render(&b, 0)
	}
	return b.String()
}

func (c *Comment) render(b *strings.Builder, depth int) {
	b.WriteString(strings.Repeat("\t", depth))
	b.WriteByte('#')
	b.WriteString(c.Text)
	b.WriteByte('\n')
}

func (*Blank) render(b *strings.Builder, _ int) {
	b.WriteByte('\n')
}

func (d *Directive) render(b *strings.Builder, depth int) {
	indent := strings.Repeat("\t", depth)
	b.WriteString(indent)
	if d.Site {
		for i, addr := range d.Tokens {
			if i > 0 {
				b.WriteString(",\n")
				b.WriteString(indent)
			}
			b.WriteString(addr)
		}
	} else {
		for i, t := range d.Tokens {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(quote(t, indent))
		}
	}
	if d.Block {
		if len(d.Tokens) > 0 {
			b.WriteByte(' ')
		}
		b.WriteByte('{')
	}
	if d.Trailing != "" {
		b.WriteString(" #")
		b.WriteString(d.Trailing)
	}
	b.WriteByte('\n')
	if d.Block {
		for _, n := range d.Nodes {
			n.render(b, depth+1)
		}
		b.WriteString(indent)
		b.WriteString("}\n")
	}
}

// quote 按需为 token 加引号：含换行用 heredoc，含双引号用反引号，含空白或结构字符用双引号
func quote(t, indent string) string {
	switch {
	case t == "":
		return `""`
	case strings.Contains(t, "\n"):
		lines := strings.Split(t, "\n")
		for i, l := range lines {
			lines[i] = indent + "\t" + l
		}
		return "<<EOT\n" + strings.Join(lines, "\n") + "\n" + indent + "\tEOT"
	case strings.Contains(t, `"`) && !strings.Contains(t, "`"):
		return "`" + t + "`"
	case strings.ContainsAny(t, " \t\""), t == "{", t == "}", strings.HasPrefix(t, "#"):
		return `"` + strings.ReplaceAll(t, `"`, `\"`) + `"`
	}
	return t
}

// ========== 查询与修改 ==========

// Name 指令名，nil 安全
func (d *Directive) Name() string {
	if d == nil || len(d.Tokens) == 0 {
		return ""
	}
	return d.Tokens[0]
}

// Args 指令参数
func (d *Directive) Args() []string {
	if len(d.Tokens) <= 1 {
		return []string{}
	}
	return d.Tokens[1:]
}

// Arg 第 i 个参数，nil 或越界返回空串，便于链式取值
func (d *Directive) Arg(i int) string {
	if d == nil || i+1 >= len(d.Tokens) {
		return ""
	}
	return d.Tokens[i+1]
}

// Append 追加节点
func (l *nodeList) Append(nodes ...Node) {
	l.Nodes = append(l.Nodes, nodes...)
}

// Directive 返回首个匹配名称的指令
func (l *nodeList) Directive(name string) *Directive {
	for _, n := range l.Nodes {
		if d, ok := n.(*Directive); ok && d.Name() == name {
			return d
		}
	}
	return nil
}

// Directives 返回所有匹配名称的指令
func (l *nodeList) Directives(name string) []*Directive {
	var out []*Directive
	for _, n := range l.Nodes {
		if d, ok := n.(*Directive); ok && d.Name() == name {
			out = append(out, d)
		}
	}
	return out
}

// All 返回全部指令
func (l *nodeList) All() []*Directive {
	var out []*Directive
	for _, n := range l.Nodes {
		if d, ok := n.(*Directive); ok {
			out = append(out, d)
		}
	}
	return out
}

// Add 追加一条指令
func (l *nodeList) Add(tokens ...string) *Directive {
	d := &Directive{Tokens: tokens}
	l.Nodes = append(l.Nodes, d)
	return d
}

// AddBlock 追加一个带块的指令
func (l *nodeList) AddBlock(tokens ...string) *Directive {
	d := &Directive{Tokens: tokens, Block: true}
	l.Nodes = append(l.Nodes, d)
	return d
}

// AddSite 追加一个站点块
func (l *nodeList) AddSite(addresses ...string) *Directive {
	d := &Directive{Tokens: addresses, Block: true, Site: true}
	l.Nodes = append(l.Nodes, d)
	return d
}

// Remove 删除所有匹配名称的指令
func (l *nodeList) Remove(name string) {
	kept := l.Nodes[:0]
	for _, n := range l.Nodes {
		if d, ok := n.(*Directive); ok && d.Name() == name {
			continue
		}
		kept = append(kept, n)
	}
	l.Nodes = kept
}

// Site 返回首个站点块
func (l *nodeList) Site() *Directive {
	if sites := l.Sites(); len(sites) > 0 {
		return sites[0]
	}
	return nil
}

// Sites 返回全部站点块
func (l *nodeList) Sites() []*Directive {
	var out []*Directive
	for _, n := range l.Nodes {
		if d, ok := n.(*Directive); ok && d.Site {
			out = append(out, d)
		}
	}
	return out
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
