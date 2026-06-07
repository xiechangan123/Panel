package apache

import (
	"slices"
	"strings"
)

const defaultIndent = "    " // 4 空格缩进

// Export 保序导出（用于片段）：按节点原有顺序输出，规范缩进
func (c *Config) Export() string {
	var b strings.Builder
	writeNodes(&b, c.Nodes, 0, false)
	return strings.TrimRight(b.String(), "\n")
}

// Render 规范化导出（用于主文件 Save）：按 order 表稳定排序后输出
func (c *Config) Render() string {
	var b strings.Builder
	writeNodes(&b, c.Nodes, 0, true)
	return strings.TrimRight(b.String(), "\n")
}

// writeNodes 递归输出节点列表；sorted 为 true 时按 order 排序并吸附注释
func writeNodes(b *strings.Builder, nodes []Node, depth int, sorted bool) {
	if sorted {
		nodes = sortNodes(nodes)
	}
	for _, n := range nodes {
		writeNode(b, n, depth, sorted)
	}
}

// writeNode 输出单个节点
func writeNode(b *strings.Builder, n Node, depth int, sorted bool) {
	indent := strings.Repeat(defaultIndent, depth)
	switch v := n.(type) {
	case *Comment:
		b.WriteString(indent)
		b.WriteByte('#')
		b.WriteString(v.Text)
		b.WriteByte('\n')
	case *Directive:
		b.WriteString(indent)
		b.WriteString(v.Name)
		for _, a := range v.Args {
			b.WriteByte(' ')
			b.WriteString(a.text())
		}
		b.WriteByte('\n')
	case *Block:
		b.WriteString(indent)
		b.WriteByte('<')
		b.WriteString(v.Name)
		for _, a := range v.Args {
			b.WriteByte(' ')
			b.WriteString(a.text())
		}
		b.WriteString(">\n")
		writeNodes(b, v.Nodes, depth+1, sorted)
		b.WriteString(indent)
		b.WriteString("</")
		b.WriteString(v.Name)
		b.WriteString(">\n")
	}
}

// sortNodes 按 order 表稳定排序，注释吸附到其后的非注释节点
func sortNodes(nodes []Node) []Node {
	type group struct {
		lead []Node // 前导连续注释
		main Node   // 非注释节点
		key  int
	}
	var groups []group
	var pending []Node
	for _, n := range nodes {
		if _, ok := n.(*Comment); ok {
			pending = append(pending, n)
			continue
		}
		groups = append(groups, group{lead: pending, main: n, key: orderOf(n)})
		pending = nil
	}
	slices.SortStableFunc(groups, func(a, b group) int { return a.key - b.key })

	out := make([]Node, 0, len(nodes))
	for _, g := range groups {
		out = append(out, g.lead...)
		out = append(out, g.main)
	}
	out = append(out, pending...) // 末尾孤立注释保留在最后
	return out
}

// orderLower 是 order 表的小写键副本，用于大小写不敏感查找
var orderLower = func() map[string]int {
	m := make(map[string]int, len(order))
	for k, v := range order {
		m[strings.ToLower(k)] = v
	}
	return m
}()

// orderOf 返回节点的排序键，未知指令排在已知功能指令之后、容器之前
func orderOf(n Node) int {
	var name string
	switch v := n.(type) {
	case *Directive:
		name = v.Name
	case *Block:
		name = v.Name
	}
	if k, ok := orderLower[strings.ToLower(name)]; ok {
		return k
	}
	return orderDefault
}
