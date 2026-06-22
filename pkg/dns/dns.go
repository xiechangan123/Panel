// Package dns 提供 DNS 配置管理功能
package dns

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/acepanel/panel/v3/pkg/io"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
)

// Manager 定义了 DNS 管理的类型
type Manager int

const (
	ManagerUnknown Manager = iota
	ManagerNetworkManager
	ManagerNetplan
	ManagerResolvConf
)

// String 返回 Manager 的字符串表示
func (m Manager) String() string {
	switch m {
	case ManagerNetworkManager:
		return "NetworkManager"
	case ManagerNetplan:
		return "netplan"
	case ManagerResolvConf:
		return "resolv.conf"
	default:
		return "unknown"
	}
}

const resolvConfPath = "/etc/resolv.conf"

var (
	nameserverRe = regexp.MustCompile(`nameserver\s+(\S+)`)

	netplanGlobs = []string{"/etc/netplan/*.yaml", "/etc/netplan/*.yml"}

	// virtualIfacePrefixes 用于排除回环和各类虚拟网络接口
	virtualIfacePrefixes = []string{
		"lo", "docker", "veth", "br-", "virbr", "vnet",
		"tun", "tap", "flannel", "cni", "cali",
	}
)

// DetectManager 检测当前系统使用的 DNS 管理方式
func DetectManager() Manager {
	if active, _ := systemctl.Status("NetworkManager"); active {
		return ManagerNetworkManager
	}
	if isNetplanAvailable() {
		return ManagerNetplan
	}
	return ManagerResolvConf
}

// GetDNS 获取当前 DNS 配置
func GetDNS() ([]string, Manager, error) {
	manager := DetectManager()

	var (
		dns []string
		err error
	)
	switch manager {
	case ManagerNetworkManager:
		dns, err = getDNSFromNetworkManager()
	case ManagerNetplan:
		dns, err = getDNSFromNetplan()
	case ManagerUnknown, ManagerResolvConf:
		// 直接走下方的 resolv.conf 读取
	}

	// 配置源读取失败或为空时回退到 resolv.conf
	if err != nil || len(dns) == 0 {
		dns, err = getDNSFromResolvConf()
	}
	return dns, manager, err
}

// SetDNS 设置 DNS 服务器，主路径失败时回退到直接写 resolv.conf
func SetDNS(dns1, dns2 string) error {
	switch DetectManager() {
	case ManagerNetworkManager:
		if err := setDNSWithNetworkManager(dns1, dns2); err == nil {
			return nil
		}
	case ManagerNetplan:
		if err := setDNSWithNetplan(dns1, dns2); err == nil {
			return nil
		}
	case ManagerUnknown, ManagerResolvConf:
		// 直接走 resolv.conf 写入
	}
	return setDNSWithResolvConf(dns1, dns2)
}

// isNetplanAvailable 检查 netplan 是否可用
func isNetplanAvailable() bool {
	if _, err := exec.LookPath("netplan"); err != nil {
		return false
	}
	for _, pattern := range netplanGlobs {
		if files, _ := filepath.Glob(pattern); len(files) > 0 {
			return true
		}
	}
	return false
}

// getDNSFromResolvConf 从 /etc/resolv.conf 获取 DNS
func getDNSFromResolvConf() ([]string, error) {
	raw, err := io.Read(resolvConfPath)
	if err != nil {
		return nil, err
	}

	matches := nameserverRe.FindAllStringSubmatch(raw, -1)
	dns := make([]string, 0, len(matches))
	for _, m := range matches {
		dns = append(dns, m[1])
	}
	return dns, nil
}

// getDNSFromNetworkManager 从 NetworkManager 收集所有设备的 DNS
func getDNSFromNetworkManager() ([]string, error) {
	output, err := shell.Execf("nmcli -t -f IP4.DNS device show")
	if err != nil {
		return nil, err
	}

	var (
		dns  []string
		seen = make(map[string]bool)
	)
	for line := range strings.SplitSeq(output, "\n") {
		// 格式: IP4.DNS[1]:8.8.8.8
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "IP4.DNS") {
			continue
		}
		_, addr, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		addr = strings.TrimSpace(addr)
		if addr == "" || seen[addr] {
			continue
		}
		seen[addr] = true
		dns = append(dns, addr)
	}
	return dns, nil
}

// getDNSFromNetplan 从 netplan 配置文件获取 DNS
func getDNSFromNetplan() ([]string, error) {
	configPath, err := findNetplanConfig()
	if err != nil {
		return nil, err
	}

	content, err := io.Read(configPath)
	if err != nil {
		return nil, err
	}

	var config netplanConfig
	if err = yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil, err
	}

	var (
		dns  []string
		seen = make(map[string]bool)
	)
	config.Network.eachInterface(func(iface *netplanInterface) {
		if iface.Nameservers == nil {
			return
		}
		for _, addr := range iface.Nameservers.Addresses {
			if seen[addr] {
				continue
			}
			seen[addr] = true
			dns = append(dns, addr)
		}
	})
	return dns, nil
}

// setDNSWithNetworkManager 使用 NetworkManager 为所有活动连接设置 DNS
func setDNSWithNetworkManager(dns1, dns2 string) error {
	connections, err := getActiveNMConnections()
	if err != nil {
		return err
	}

	dnsServers := dns1
	if dns2 != "" {
		dnsServers = dns1 + "," + dns2
	}

	var (
		lastErr error
		success int
	)
	for _, conn := range connections {
		if _, err := shell.Execf("nmcli connection modify %s ipv4.dns %s", conn, dnsServers); err != nil {
			lastErr = fmt.Errorf("set DNS for connection %s: %w", conn, err)
			continue
		}
		// 确保自定义 DNS 优先并忽略 DHCP 下发的 DNS
		_, _ = shell.Execf("nmcli connection modify %s ipv4.dns-priority -1", conn)
		_, _ = shell.Execf("nmcli connection modify %s ipv4.ignore-auto-dns yes", conn)
		if _, err := shell.Execf("nmcli connection up %s", conn); err != nil {
			lastErr = fmt.Errorf("reactivate connection %s: %w", conn, err)
			continue
		}
		success++
	}

	if success == 0 {
		return lastErr
	}
	return nil
}

// getActiveNMConnections 返回所有活动的 NetworkManager 连接名（已加 shell 单引号）
func getActiveNMConnections() ([]string, error) {
	output, err := shell.Execf("nmcli -t -f NAME,DEVICE connection show --active")
	if err != nil {
		return nil, err
	}

	var names []string
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 格式: NAME:DEVICE
		name, device, ok := strings.Cut(line, ":")
		if !ok || device == "" || !isValidNetworkInterface(device) {
			continue
		}
		names = append(names, shellQuote(name))
	}

	if len(names) == 0 {
		return nil, errors.New("no active NetworkManager connections found")
	}
	return names, nil
}

// shellQuote 为 shell 参数加单引号，处理参数中包含单引号的情况
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// setDNSWithNetplan 使用 netplan 设置 DNS
func setDNSWithNetplan(dns1, dns2 string) error {
	configPath, err := findNetplanConfig()
	if err != nil {
		return err
	}

	content, err := io.Read(configPath)
	if err != nil {
		return err
	}

	newContent, err := updateNetplanDNS(content, dns1, dns2)
	if err != nil {
		return err
	}

	if err = io.Write(configPath, newContent, 0644); err != nil {
		return fmt.Errorf("write netplan config: %w", err)
	}
	if _, err = shell.Execf("netplan apply"); err != nil {
		return fmt.Errorf("apply netplan config: %w", err)
	}
	return nil
}

// findNetplanConfig 查找最后一个 netplan 配置文件，netplan 按字母顺序处理且后者覆盖前者
func findNetplanConfig() (string, error) {
	for _, pattern := range netplanGlobs {
		files, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		if len(files) > 0 {
			return files[len(files)-1], nil
		}
	}
	return "", errors.New("netplan config file not found")
}

// netplanConfig netplan 配置结构
type netplanConfig struct {
	Network netplanNetwork `yaml:"network"`
}

// netplanNetwork 网络配置
type netplanNetwork struct {
	Version   int                          `yaml:"version,omitempty"`
	Renderer  string                       `yaml:"renderer,omitempty"`
	Ethernets map[string]*netplanInterface `yaml:"ethernets,omitempty"`
	Wifis     map[string]*netplanInterface `yaml:"wifis,omitempty"`
	Bonds     map[string]*netplanInterface `yaml:"bonds,omitempty"`
	Bridges   map[string]*netplanInterface `yaml:"bridges,omitempty"`
	Vlans     map[string]*netplanInterface `yaml:"vlans,omitempty"`
}

// eachInterface 遍历所有非空网络接口
func (n *netplanNetwork) eachInterface(fn func(*netplanInterface)) {
	groups := []map[string]*netplanInterface{
		n.Ethernets, n.Wifis, n.Bonds, n.Bridges, n.Vlans,
	}
	for _, group := range groups {
		for _, iface := range group {
			if iface != nil {
				fn(iface)
			}
		}
	}
}

// netplanInterface 网络接口配置
type netplanInterface struct {
	Nameservers *netplanNameservers  `yaml:"nameservers,omitempty"`
	Extra       map[string]yaml.Node `yaml:",inline"`
}

// netplanNameservers DNS 配置
type netplanNameservers struct {
	Addresses []string `yaml:"addresses,omitempty"`
	Search    []string `yaml:"search,omitempty"`
}

// updateNetplanDNS 更新 netplan 配置中的 DNS
func updateNetplanDNS(content, dns1, dns2 string) (string, error) {
	var config netplanConfig
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return "", fmt.Errorf("parse netplan config: %w", err)
	}

	addresses := []string{dns1}
	if dns2 != "" {
		addresses = append(addresses, dns2)
	}

	updated := false
	config.Network.eachInterface(func(iface *netplanInterface) {
		if iface.Nameservers == nil {
			iface.Nameservers = &netplanNameservers{}
		}
		iface.Nameservers.Addresses = addresses
		updated = true
	})

	// 配置中没有任何接口时为活动接口创建条目
	if !updated {
		iface := detectActiveInterface()
		if iface == "" {
			return "", errors.New("no network interface found in config and failed to detect active interface")
		}
		if config.Network.Ethernets == nil {
			config.Network.Ethernets = make(map[string]*netplanInterface)
		}
		config.Network.Ethernets[iface] = &netplanInterface{
			Nameservers: &netplanNameservers{Addresses: addresses},
		}
		if config.Network.Version == 0 {
			config.Network.Version = 2
		}
	}

	output, err := yaml.Marshal(&config)
	if err != nil {
		return "", fmt.Errorf("marshal netplan config: %w", err)
	}
	return string(output), nil
}

// detectActiveInterface 检测当前活动的物理网络接口，优先返回默认路由出口接口
func detectActiveInterface() string {
	// 优先解析默认路由的出口接口
	if output, err := shell.Execf("ip route show default"); err == nil {
		for line := range strings.SplitSeq(output, "\n") {
			// 格式: default via 192.168.1.1 dev eth0 proto dhcp metric 100
			fields := strings.Fields(line)
			for i, f := range fields {
				if f == "dev" && i+1 < len(fields) && isValidNetworkInterface(fields[i+1]) {
					return fields[i+1]
				}
			}
		}
	}

	// 回退：在系统接口中查找首个有效的 UP 接口
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		if !isValidNetworkInterface(iface.Name) {
			continue
		}
		if addrs, _ := iface.Addrs(); len(addrs) > 0 {
			return iface.Name
		}
	}
	return ""
}

// isValidNetworkInterface 检查接口名是否为有效的物理/外部网络接口
func isValidNetworkInterface(name string) bool {
	if name == "" {
		return false
	}
	for _, prefix := range virtualIfacePrefixes {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	return true
}

// setDNSWithResolvConf 直接修改 /etc/resolv.conf 设置 DNS
func setDNSWithResolvConf(dns1, dns2 string) error {
	var b strings.Builder
	b.WriteString("nameserver " + dns1 + "\n")
	if dns2 != "" {
		b.WriteString("nameserver " + dns2 + "\n")
	}
	if err := io.Write(resolvConfPath, b.String(), 0644); err != nil {
		return fmt.Errorf("write %s: %w", resolvConfPath, err)
	}
	return nil
}
