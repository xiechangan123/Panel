package nginx

import (
	"strings"

	"github.com/acepanel/panel/v3/pkg/webserver/conf"
)

const indentUnit = "    "

// Export 保序渲染为配置文本，顶层块之间空一行
func Export(c *conf.Config) string {
	var b strings.Builder
	writeNodes(&b, c.Nodes, 0, false)
	return b.String()
}

// Render 按 order 表排序后渲染，带块的指令排在普通指令之后，用于站点主文件
func Render(c *conf.Config) string {
	var b strings.Builder
	writeNodes(&b, c.Nodes, 0, true)
	return b.String()
}

func writeNodes(b *strings.Builder, nodes []conf.Node, depth int, sorted bool) {
	if sorted {
		nodes = conf.Sort(nodes, orderOf)
	}
	indent := strings.Repeat(indentUnit, depth)
	for i, n := range nodes {
		switch v := n.(type) {
		case *conf.Comment:
			b.WriteString(indent + "#" + v.Text + "\n")
		case *conf.Blank:
			b.WriteByte('\n')
		case *conf.Directive:
			if depth == 0 && i > 0 && v.Block != nil {
				if _, blank := nodes[i-1].(*conf.Blank); !blank {
					b.WriteByte('\n')
				}
			}
			writeDirective(b, v, depth, sorted)
		}
	}
}

func writeDirective(b *strings.Builder, d *conf.Directive, depth int, sorted bool) {
	indent := strings.Repeat(indentUnit, depth)
	b.WriteString(indent + quote(conf.Arg{Value: d.Name}))
	raw, hasRaw := "", false
	for _, a := range d.Args {
		if a.Quote == conf.QuoteRaw {
			raw, hasRaw = a.Value, true
			continue
		}
		// if 条件的右括号紧贴前一个参数，如 if ($host = "a.com")
		if a.Value == ")" && a.Quote != conf.QuoteDouble && a.Quote != conf.QuoteSingle {
			b.WriteString(")")
			continue
		}
		b.WriteString(" " + quote(a))
	}
	switch {
	case d.Block != nil:
		b.WriteString(" {")
		writeTrailing(b, d.Trailing)
		b.WriteByte('\n')
		writeNodes(b, d.Nodes, depth+1, sorted)
		b.WriteString(indent + "}\n")
	case hasRaw:
		b.WriteString(" {" + raw + "}")
		writeTrailing(b, d.Trailing)
		b.WriteByte('\n')
	default:
		b.WriteByte(';')
		writeTrailing(b, d.Trailing)
		b.WriteByte('\n')
	}
}

func writeTrailing(b *strings.Builder, trailing string) {
	if trailing != "" {
		b.WriteString(" #" + trailing)
	}
}

// quote 按引号风格输出参数，自动模式下含空白、结构字符或引号时加引号，值里有双引号则改用单引号
func quote(a conf.Arg) string {
	q := a.Quote
	if q == conf.QuoteAuto {
		switch {
		case a.Value == "" || strings.ContainsAny(a.Value, " \t\r\n;{}#'\"\\"):
			q = conf.QuoteDouble
			if strings.Contains(a.Value, `"`) && !strings.Contains(a.Value, `'`) {
				q = conf.QuoteSingle
			}
		default:
			return a.Value
		}
	}
	switch q {
	case conf.QuoteDouble:
		return `"` + escape(a.Value, '"') + `"`
	case conf.QuoteSingle:
		return `'` + escape(a.Value, '\'') + `'`
	default:
		return a.Value
	}
}

// escape 与解析对称：反斜杠、引号与控制符转义
func escape(s string, quote byte) string {
	var b strings.Builder
	for i := range len(s) {
		switch ch := s[i]; ch {
		case '\\', quote:
			b.WriteByte('\\')
			b.WriteByte(ch)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

// orderOf 排序键：未知指令排在已知指令之后，带块的指令再靠后
func orderOf(d *conf.Directive) int {
	key, ok := order[d.Name]
	if !ok {
		key = orderDefault
	}
	if d.Block != nil {
		key += orderBlock
	}
	return key
}
