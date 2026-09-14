package conf

import "slices"

// Sort 按 key 稳定排序指令，注释与空行吸附到其后的指令一起移动，末尾孤立的注释保留在最后
func Sort(nodes []Node, key func(*Directive) int) []Node {
	type group struct {
		lead []Node
		main *Directive
		key  int
	}
	var groups []group
	var pending []Node
	for _, n := range nodes {
		d, ok := n.(*Directive)
		if !ok {
			pending = append(pending, n)
			continue
		}
		groups = append(groups, group{lead: pending, main: d, key: key(d)})
		pending = nil
	}
	slices.SortStableFunc(groups, func(a, b group) int { return a.key - b.key })

	out := make([]Node, 0, len(nodes))
	for _, g := range groups {
		out = append(out, g.lead...)
		out = append(out, g.main)
	}
	return append(out, pending...)
}
