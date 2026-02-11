package firewall

import (
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
