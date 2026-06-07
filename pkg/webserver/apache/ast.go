package apache

import "strings"

// QuoteStyle 表示参数原始的引号风格，用于忠实复现
type QuoteStyle uint8

const (
	QuoteNone   QuoteStyle = iota // 无引号
	QuoteDouble                   // 双引号
	QuoteSingle                   // 单引号
)

// Argument 是一个指令或块参数：解引号后的纯值 + 原始引号风格
type Argument struct {
	Value string     // 解引号后的真实值，如 www.example.com
	Quote QuoteStyle // 复现时据此加回引号
}

// String 返回参数的纯值，便于直接当字符串使用
func (a Argument) String() string { return a.Value }

// text 按引号风格复现带引号的文本
func (a Argument) text() string {
	switch a.Quote {
	case QuoteDouble:
		return `"` + strings.ReplaceAll(a.Value, `"`, `\"`) + `"`
	case QuoteSingle:
		return "'" + a.Value + "'"
	default:
		return a.Value
	}
}

// needsQuote 判断裸值是否必须加双引号才能安全写入配置
func needsQuote(s string) bool {
	if s == "" {
		return true
	}
	return strings.ContainsAny(s, " \t\"<>")
}

// arg 由裸字符串构造参数，自动决定是否加引号
func arg(value string) Argument {
	if needsQuote(value) {
		return Argument{Value: value, Quote: QuoteDouble}
	}
	return Argument{Value: value, Quote: QuoteNone}
}

// dquote 构造一个强制双引号的参数，用于值含特殊编码必须加引号的指令（如 Substitute）
func dquote(value string) Argument {
	return Argument{Value: value, Quote: QuoteDouble}
}

// argsOf 批量构造参数
func argsOf(values ...string) []Argument {
	if len(values) == 0 {
		return nil
	}
	out := make([]Argument, len(values))
	for i, v := range values {
		out[i] = arg(v)
	}
	return out
}

// argValues 提取参数的纯值切片
func argValues(args []Argument) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = a.Value
	}
	return out
}

// Node 是配置中可出现在容器内的元素：指令、块或注释
type Node interface {
	node()
}

// nodeList 是有序节点容器，被 Config 和 Block 嵌入以共享查询/修改方法
type nodeList struct {
	Nodes []Node
}

// Config 是一个配置单元（apache.conf 或片段文件）的根节点
type Config struct {
	nodeList
}

// Directive 是普通指令，独占一行，如 ServerName a b
type Directive struct {
	Name string
	Args []Argument
}

// Block 是容器块，如 <Directory /x> ... </Directory>，可任意深度嵌套
// VirtualHost 即 Name=="VirtualHost" 的 Block，不再是独立类型
type Block struct {
	Name     string
	Args     []Argument
	nodeList // 子节点（Nodes 字段），嵌套块就在这里
}

// Comment 是整行注释，Text 为 # 之后的原文（含前导空格）
type Comment struct {
	Text string
}

func (*Directive) node() {}
func (*Block) node()     {}
func (*Comment) node()   {}

// Dir 构造一条指令，参数自动判断引号
func Dir(name string, args ...string) *Directive {
	return &Directive{Name: name, Args: argsOf(args...)}
}

// Blk 构造一个块，参数自动判断引号；用 Add 填充子节点
func Blk(name string, args ...string) *Block {
	return &Block{Name: name, Args: argsOf(args...)}
}

// Cmt 构造一条注释，传入不含 # 的纯文本
func Cmt(text string) *Comment {
	return &Comment{Text: " " + text}
}

// Append 向容器追加节点
func (l *nodeList) Append(nodes ...Node) {
	l.Nodes = append(l.Nodes, nodes...)
}

// Append 向块追加子节点，返回自身以便链式调用
func (b *Block) Append(nodes ...Node) *Block {
	b.Nodes = append(b.Nodes, nodes...)
	return b
}
