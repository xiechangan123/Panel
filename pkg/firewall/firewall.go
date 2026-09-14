package firewall

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/samber/lo"

	"github.com/acepanel/panel/v3/pkg/shell"
)

// Firewall 防火墙统一接口
// 变更方法由 NewFirewall 返回的实现统一加锁并断开取消链，各后端实现内无需再处理
type Firewall interface {
	// Status 获取防火墙运行状态
	Status(ctx context.Context) (bool, error)
	// Enable 启用防火墙
	Enable(ctx context.Context) error
	// Disable 禁用防火墙
	Disable(ctx context.Context) error

	// ListRule 列出所有规则
	ListRule(ctx context.Context) ([]FireInfo, error)
	// Port 添加/删除端口规则
	Port(ctx context.Context, rule FireInfo, operation Operation) error
	// RichRules 添加/删除富规则（IP/高级规则）
	RichRules(ctx context.Context, rule FireInfo, operation Operation) error

	// ListForward 列出所有转发规则
	ListForward(ctx context.Context) ([]FireForwardInfo, error)
	// Forward 添加/删除转发规则
	Forward(ctx context.Context, rule Forward, operation Operation) error

	// PingStatus 获取 Ping 状态（true 为允许）
	PingStatus(ctx context.Context) (bool, error)
	// UpdatePingStatus 更新 Ping 状态
	UpdatePingStatus(ctx context.Context, status bool) error
}

// NewFirewall 返回当前系统的防火墙实现
func NewFirewall() Firewall {
	return lockedFirewall{}
}

// detectFirewall 防火墙类型是开机后不再变化的环境事实，探测一次后全局复用
// 探测用不可取消 ctx：跟随调用方 ctx 时，取消会让两次探测都失败而退回 firewalld 默认值，并把这个错误结论缓存到进程结束
var detectFirewall = sync.OnceValue(func() Firewall {
	ctx := context.Background()
	if _, err := shell.Execf(ctx, "firewall-cmd --version"); err == nil {
		return newFirewalld()
	}
	if _, err := shell.Execf(ctx, "ufw version"); err == nil {
		return newUFW()
	}
	// 默认 firewalld
	return newFirewalld()
})

// mu 串行化所有防火墙变更操作
// ufw 在加锁前就已将 user.rules 读入内存，并发执行时后写入的进程会覆盖先写入的规则且都返回成功
var mu sync.Mutex

// lockedFirewall 变更操作加锁并断开取消链，读操作直接透传
// 每个变更都是「写永久配置 + reload」的组合，中途取消会让两者脱节：
// firewalld 下规则进了永久配置却没进 runtime，ufw 下面板读的规则文件已改而内核里的规则还在跑
type lockedFirewall struct{}

func (lockedFirewall) Status(ctx context.Context) (bool, error) {
	return detectFirewall().Status(ctx)
}

func (lockedFirewall) ListRule(ctx context.Context) ([]FireInfo, error) {
	return detectFirewall().ListRule(ctx)
}

func (lockedFirewall) ListForward(ctx context.Context) ([]FireForwardInfo, error) {
	return detectFirewall().ListForward(ctx)
}

func (lockedFirewall) PingStatus(ctx context.Context) (bool, error) {
	return detectFirewall().PingStatus(ctx)
}

func (lockedFirewall) Enable(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()
	return detectFirewall().Enable(context.WithoutCancel(ctx))
}

func (lockedFirewall) Disable(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()
	return detectFirewall().Disable(context.WithoutCancel(ctx))
}

func (lockedFirewall) Port(ctx context.Context, rule FireInfo, operation Operation) error {
	mu.Lock()
	defer mu.Unlock()
	return detectFirewall().Port(context.WithoutCancel(ctx), rule, operation)
}

func (lockedFirewall) RichRules(ctx context.Context, rule FireInfo, operation Operation) error {
	mu.Lock()
	defer mu.Unlock()
	return detectFirewall().RichRules(context.WithoutCancel(ctx), rule, operation)
}

func (lockedFirewall) Forward(ctx context.Context, rule Forward, operation Operation) error {
	mu.Lock()
	defer mu.Unlock()
	return detectFirewall().Forward(context.WithoutCancel(ctx), rule, operation)
}

func (lockedFirewall) UpdatePingStatus(ctx context.Context, status bool) error {
	mu.Lock()
	defer mu.Unlock()
	return detectFirewall().UpdatePingStatus(context.WithoutCancel(ctx), status)
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
			grouped[key] = new(r)
			order = append(order, key)
		}
	}

	return lo.Map(order, func(key string, _ int) FireInfo {
		return *grouped[key]
	})
}
