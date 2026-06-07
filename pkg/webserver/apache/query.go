package apache

import "strings"

// 以下查询/修改方法挂在 *nodeList 上，Config 与 Block 通过嵌入共享，名称比较一律大小写不敏感

// Get 返回首个匹配名称的指令
func (l *nodeList) Get(name string) *Directive {
	for _, n := range l.Nodes {
		if d, ok := n.(*Directive); ok && strings.EqualFold(d.Name, name) {
			return d
		}
	}
	return nil
}

// GetAll 返回所有匹配名称的指令
func (l *nodeList) GetAll(name string) []*Directive {
	var out []*Directive
	for _, n := range l.Nodes {
		if d, ok := n.(*Directive); ok && strings.EqualFold(d.Name, name) {
			out = append(out, d)
		}
	}
	return out
}

// Has 判断是否存在指定指令
func (l *nodeList) Has(name string) bool {
	return l.Get(name) != nil
}

// Value 返回指令首个参数值，不存在返回空串
func (l *nodeList) Value(name string) string {
	if d := l.Get(name); d != nil && len(d.Args) > 0 {
		return d.Args[0].Value
	}
	return ""
}

// Values 返回指令全部参数值
func (l *nodeList) Values(name string) []string {
	if d := l.Get(name); d != nil {
		return argValues(d.Args)
	}
	return nil
}

// Add 追加一条指令，参数自动判断引号，返回新指令
func (l *nodeList) Add(name string, args ...string) *Directive {
	d := Dir(name, args...)
	l.Nodes = append(l.Nodes, d)
	return d
}

// Set 设置指令：存在则更新参数，否则追加
func (l *nodeList) Set(name string, args ...string) *Directive {
	if d := l.Get(name); d != nil {
		d.Args = argsOf(args...)
		return d
	}
	return l.Add(name, args...)
}

// Remove 删除首个匹配名称的指令，返回是否删除
func (l *nodeList) Remove(name string) bool {
	for i, n := range l.Nodes {
		if d, ok := n.(*Directive); ok && strings.EqualFold(d.Name, name) {
			l.Nodes = append(l.Nodes[:i], l.Nodes[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveAll 删除所有匹配名称的指令，返回删除数量
func (l *nodeList) RemoveAll(name string) int {
	return l.RemoveFunc(name, func(*Directive) bool { return true })
}

// RemoveFunc 删除满足谓词的匹配名称指令，返回删除数量
func (l *nodeList) RemoveFunc(name string, pred func(*Directive) bool) int {
	kept := l.Nodes[:0]
	count := 0
	for _, n := range l.Nodes {
		if d, ok := n.(*Directive); ok && strings.EqualFold(d.Name, name) && pred(d) {
			count++
			continue
		}
		kept = append(kept, n)
	}
	l.Nodes = kept
	return count
}

// GetBlock 返回首个匹配类型（及可选参数）的块
func (l *nodeList) GetBlock(name string, args ...string) *Block {
	for _, n := range l.Nodes {
		if b, ok := n.(*Block); ok && strings.EqualFold(b.Name, name) {
			if len(args) == 0 || argsEqual(b.Args, args) {
				return b
			}
		}
	}
	return nil
}

// AddBlock 追加一个块，返回新块
func (l *nodeList) AddBlock(name string, args ...string) *Block {
	b := Blk(name, args...)
	l.Nodes = append(l.Nodes, b)
	return b
}

// VirtualHosts 返回所有 VirtualHost 块
func (l *nodeList) VirtualHosts() []*Block {
	var out []*Block
	for _, n := range l.Nodes {
		if b, ok := n.(*Block); ok && strings.EqualFold(b.Name, "VirtualHost") {
			out = append(out, b)
		}
	}
	return out
}

// AddVirtualHost 追加一个 VirtualHost 块
func (l *nodeList) AddVirtualHost(args ...string) *Block {
	return l.AddBlock("VirtualHost", args...)
}

// Find 按点路径查找指令，跨越嵌套块，如 Find("IfModule.Proxy.BalancerMember")
func (l *nodeList) Find(path string) []*Directive {
	return findDirectives(l.Nodes, strings.Split(path, "."))
}

// FindOne 返回点路径命中的首个指令，无命中返回 nil
func (l *nodeList) FindOne(path string) *Directive {
	if ds := l.Find(path); len(ds) > 0 {
		return ds[0]
	}
	return nil
}

// FindBlocks 按点路径查找块，如 FindBlocks("IfModule.Proxy")
func (l *nodeList) FindBlocks(path string) []*Block {
	return findBlocks(l.Nodes, strings.Split(path, "."))
}

// SetArgs 替换块参数，自动判断引号
func (b *Block) SetArgs(args ...string) {
	b.Args = argsOf(args...)
}

// AppendArg 追加一个块参数
func (b *Block) AppendArg(value string) {
	b.Args = append(b.Args, arg(value))
}

// ArgValues 返回块参数的纯值切片
func (b *Block) ArgValues() []string {
	return argValues(b.Args)
}

// findDirectives 沿点路径递归查找指令
func findDirectives(nodes []Node, parts []string) []*Directive {
	if len(parts) == 0 {
		return nil
	}
	head := parts[0]
	if len(parts) == 1 {
		var out []*Directive
		for _, n := range nodes {
			if d, ok := n.(*Directive); ok && strings.EqualFold(d.Name, head) {
				out = append(out, d)
			}
		}
		return out
	}
	var out []*Directive
	for _, n := range nodes {
		if b, ok := n.(*Block); ok && strings.EqualFold(b.Name, head) {
			out = append(out, findDirectives(b.Nodes, parts[1:])...)
		}
	}
	return out
}

// findBlocks 沿点路径递归查找块
func findBlocks(nodes []Node, parts []string) []*Block {
	if len(parts) == 0 {
		return nil
	}
	head := parts[0]
	last := len(parts) == 1
	var out []*Block
	for _, n := range nodes {
		b, ok := n.(*Block)
		if !ok || !strings.EqualFold(b.Name, head) {
			continue
		}
		if last {
			out = append(out, b)
		} else {
			out = append(out, findBlocks(b.Nodes, parts[1:])...)
		}
	}
	return out
}

// argsEqual 比较参数值与字符串切片是否逐项相等
func argsEqual(args []Argument, want []string) bool {
	if len(args) != len(want) {
		return false
	}
	for i, a := range args {
		if a.Value != want[i] {
			return false
		}
	}
	return true
}
