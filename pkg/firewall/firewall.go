package firewall

import (
	"fmt"
	"net"
	"strings"

	"github.com/acepanel/panel/pkg/shell"
)

// Firewall 防火墙统一接口
type Firewall interface {
	// Status 获取防火墙运行状态
	Status() (bool, error)
	// Enable 启用防火墙
	Enable() error
	// Disable 禁用防火墙
	Disable() error

	// ListRule 列出所有规则
	ListRule() ([]FireInfo, error)
	// Port 添加/删除端口规则
	Port(rule FireInfo, operation Operation) error
	// RichRules 添加/删除富规则（IP/高级规则）
	RichRules(rule FireInfo, operation Operation) error

	// ListForward 列出所有转发规则
	ListForward() ([]FireForwardInfo, error)
	// Forward 添加/删除转发规则
	Forward(rule Forward, operation Operation) error

	// PingStatus 获取 Ping 状态（true 为允许）
	PingStatus() (bool, error)
	// UpdatePingStatus 更新 Ping 状态
	UpdatePingStatus(status bool) error
}

// NewFirewall 自动检测系统防火墙类型并返回对应实现
func NewFirewall() Firewall {
	if _, err := shell.Execf("firewall-cmd --version"); err == nil {
		return newFirewalld()
	}
	if _, err := shell.Execf("ufw version"); err == nil {
		return newUFW()
	}
	// 默认 firewalld
	return newFirewalld()
}

// isLocalAddress 判断是否为本地地址
func isLocalAddress(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	if parsed.IsLoopback() {
		return true
	}
	if parsed.IsUnspecified() {
		return true
	}

	return false
}

// buildProtocols 拆分协议字符串为列表
func buildProtocols(protocol Protocol) []string {
	return strings.Split(string(protocol), "/")
}

// mergeRules 将同端口/同地址/同策略/同方向、仅协议不同（tcp vs udp）的规则合并为 tcp/udp
func mergeRules(rules []FireInfo) []FireInfo {
	type ruleKey struct {
		Type      Type
		Family    string
		Address   string
		PortStart uint
		PortEnd   uint
		Strategy  Strategy
		Direction Direction
	}

	grouped := make(map[string]*FireInfo)
	var order []string

	for i := range rules {
		r := rules[i]
		key := fmt.Sprintf("%s|%s|%s|%d|%d|%s|%s",
			r.Type, r.Family, r.Address, r.PortStart, r.PortEnd, r.Strategy, r.Direction)

		if existing, ok := grouped[key]; ok {
			// 合并协议：tcp + udp → tcp/udp
			if existing.Protocol != r.Protocol {
				existing.Protocol = ProtocolTCPUDP
			}
		} else {
			clone := r
			grouped[key] = &clone
			order = append(order, key)
		}
	}

	merged := make([]FireInfo, 0, len(order))
	for _, key := range order {
		merged = append(merged, *grouped[key])
	}
	return merged
}
