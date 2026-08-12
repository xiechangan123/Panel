package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
)

const (
	ManagerUnsupported    = "unsupported"
	ManagerNetworkManager = "networkmanager"
	ManagerNetplan        = "netplan"
	ManagerIfupdown       = "ifupdown"
	ModeAuto              = "auto"
	ModeManual            = "manual"
	ModeDisabled          = "disabled"
)

type FamilyConfig struct {
	Mode      string   `json:"mode"`
	Addresses []string `json:"addresses"`
	Gateway   string   `json:"gateway"`
	AutoDNS   bool     `json:"auto_dns"`
	DNS       []string `json:"dns"`
}

type Config struct {
	Name string       `json:"name"`
	MTU  int          `json:"mtu"`
	IPv4 FamilyConfig `json:"ipv4"`
	IPv6 FamilyConfig `json:"ipv6"`
}

type Interface struct {
	Name          string       `json:"name"`
	Type          string       `json:"type"`
	State         string       `json:"state"`
	MAC           string       `json:"mac"`
	CurrentMTU    int          `json:"current_mtu"`
	ConfiguredMTU int          `json:"configured_mtu"`
	CurrentIPv4   []string     `json:"current_ipv4"`
	CurrentIPv6   []string     `json:"current_ipv6"`
	Editable      bool         `json:"editable"`
	Reason        string       `json:"reason"`
	IPv4          FamilyConfig `json:"ipv4"`
	IPv6          FamilyConfig `json:"ipv6"`
}

type Result struct {
	Manager string      `json:"manager"`
	Items   []Interface `json:"items"`
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type Service struct {
	mu sync.Mutex
}

func New() *Service {
	return &Service{}
}

func (s *Service) Interfaces(ctx context.Context) (*Result, error) {
	items, err := runtimeInterfaces()
	if err != nil {
		return nil, err
	}
	manager := detectManager(ctx)
	switch manager {
	case ManagerNetworkManager:
		err = loadNetworkManager(ctx, items)
	case ManagerNetplan:
		err = loadNetplan(ctx, items)
	case ManagerIfupdown:
		err = loadIfupdown(items)
	default:
		for i := range items {
			items[i].Reason = "unsupported network manager"
		}
	}
	if err != nil {
		return nil, err
	}
	return &Result{Manager: manager, Items: items}, nil
}

func (s *Service) Update(ctx context.Context, config Config) error {
	if err := Validate(config); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.Interfaces(ctx)
	if err != nil {
		return err
	}
	index := slices.IndexFunc(result.Items, func(item Interface) bool {
		return item.Name == config.Name
	})
	if index < 0 {
		return &ValidationError{Message: "network interface does not exist or is not manageable"}
	}
	if !result.Items[index].Editable {
		return &ValidationError{Message: result.Items[index].Reason}
	}

	switch result.Manager {
	case ManagerNetworkManager:
		return updateNetworkManager(ctx, config)
	case ManagerNetplan:
		return updateNetplan(ctx, result.Items[index], config)
	case ManagerIfupdown:
		return updateIfupdown(ctx, config)
	default:
		return &ValidationError{Message: "unsupported network manager"}
	}
}

func Validate(config Config) error {
	if config.Name == "" || filepath.Base(config.Name) != config.Name || strings.ContainsAny(config.Name, " \t\r\n/") {
		return &ValidationError{Message: "invalid network interface name"}
	}
	if config.MTU != 0 && (config.MTU < 68 || config.MTU > 65535) {
		return &ValidationError{Message: "MTU must be 0 or between 68 and 65535"}
	}
	if config.IPv4.Mode == ModeDisabled && config.IPv6.Mode == ModeDisabled {
		return &ValidationError{Message: "IPv4 and IPv6 cannot both be disabled"}
	}
	if config.IPv6.Mode != ModeDisabled && config.MTU != 0 && config.MTU < 1280 {
		return &ValidationError{Message: "MTU must not be lower than 1280 when IPv6 is enabled"}
	}
	if err := validateFamily("IPv4", config.IPv4, true); err != nil {
		return err
	}
	return validateFamily("IPv6", config.IPv6, false)
}

func validateFamily(name string, config FamilyConfig, ipv4 bool) error {
	if !slices.Contains([]string{ModeAuto, ModeManual, ModeDisabled}, config.Mode) {
		return &ValidationError{Message: fmt.Sprintf("invalid %s mode", name)}
	}
	if config.Mode == ModeDisabled {
		if len(config.Addresses) > 0 || config.Gateway != "" || len(config.DNS) > 0 {
			return &ValidationError{Message: fmt.Sprintf("disabled %s configuration must be empty", name)}
		}
		return nil
	}
	if config.Mode == ModeManual && len(config.Addresses) == 0 {
		return &ValidationError{Message: fmt.Sprintf("manual %s configuration requires an address", name)}
	}

	addresses := make(map[netip.Prefix]struct{}, len(config.Addresses))
	for _, value := range config.Addresses {
		prefix, err := netip.ParsePrefix(value)
		if err != nil || prefix.Addr().Is4() != ipv4 {
			return &ValidationError{Message: fmt.Sprintf("invalid %s CIDR: %s", name, value)}
		}
		if _, exists := addresses[prefix]; exists {
			return &ValidationError{Message: fmt.Sprintf("duplicate %s address: %s", name, value)}
		}
		addresses[prefix] = struct{}{}
	}
	if config.Gateway != "" {
		gateway, err := netip.ParseAddr(config.Gateway)
		if err != nil || gateway.Is4() != ipv4 {
			return &ValidationError{Message: fmt.Sprintf("invalid %s gateway", name)}
		}
	}
	dns := make(map[netip.Addr]struct{}, len(config.DNS))
	for _, value := range config.DNS {
		server, err := netip.ParseAddr(value)
		if err != nil || server.Is4() != ipv4 {
			return &ValidationError{Message: fmt.Sprintf("invalid %s DNS: %s", name, value)}
		}
		if _, exists := dns[server]; exists {
			return &ValidationError{Message: fmt.Sprintf("duplicate %s DNS: %s", name, value)}
		}
		dns[server] = struct{}{}
	}
	return nil
}

func IsValidationError(err error) bool {
	var target *ValidationError
	return errors.As(err, &target)
}

func detectManager(ctx context.Context) string {
	if hasCommand("netplan") {
		files, _ := filepath.Glob("/etc/netplan/*.yaml")
		yml, _ := filepath.Glob("/etc/netplan/*.yml")
		if len(files)+len(yml) > 0 {
			if _, err := run(ctx, "netplan", "get"); err == nil {
				return ManagerNetplan
			}
		}
	}
	if hasCommand("nmcli") {
		if output, err := run(ctx, "nmcli", "-t", "-f", "RUNNING", "general", "status"); err == nil && strings.EqualFold(output, "running") {
			return ManagerNetworkManager
		}
	}
	if hasCommand("ifquery") {
		if _, err := os.Stat("/etc/network/interfaces"); err == nil {
			return ManagerIfupdown
		}
	}
	return ManagerUnsupported
}

func runtimeInterfaces() ([]Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	items := make([]Interface, 0, len(interfaces))
	for _, current := range interfaces {
		kind := interfaceType(current.Name)
		if kind == "" {
			continue
		}
		state := "down"
		if current.Flags&net.FlagUp != 0 {
			state = "up"
		}
		ipv4, ipv6 := currentAddresses(current)
		items = append(items, Interface{
			Name: current.Name, Type: kind, State: state, MAC: current.HardwareAddr.String(),
			CurrentMTU: current.MTU, CurrentIPv4: ipv4, CurrentIPv6: ipv6,
			IPv4: emptyFamily(), IPv6: emptyFamily(),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func interfaceType(name string) string {
	if name == "lo" {
		return ""
	}
	for _, prefix := range []string{"docker", "veth", "cni", "flannel", "cali", "virbr", "br-", "vnet", "macvlan", "macvtap", "podman", "lxc", "kube", "tun", "tap", "tailscale", "wg", "sit", "gre", "gretap", "ip6tnl", "dummy"} {
		if strings.HasPrefix(name, prefix) {
			return ""
		}
	}
	base := filepath.Join("/sys/class/net", name)
	switch {
	case exists(filepath.Join(base, "bonding")):
		return "bond"
	case exists(filepath.Join(base, "bridge")):
		return "bridge"
	case exists(filepath.Join("/proc/net/vlan", name)):
		return "vlan"
	case exists(filepath.Join(base, "device")):
		if exists(filepath.Join(base, "wireless")) {
			return "wifi"
		}
		return "ethernet"
	default:
		return ""
	}
}

func currentAddresses(current net.Interface) ([]string, []string) {
	addresses, _ := current.Addrs()
	var ipv4, ipv6 []string
	for _, address := range addresses {
		ip, _, err := net.ParseCIDR(address.String())
		if err != nil || ip.IsUnspecified() {
			continue
		}
		if ip.To4() != nil {
			ipv4 = append(ipv4, address.String())
		} else if !ip.IsLinkLocalUnicast() {
			ipv6 = append(ipv6, address.String())
		}
	}
	return unique(ipv4), unique(ipv6)
}

func emptyFamily() FamilyConfig {
	return FamilyConfig{Mode: ModeDisabled, Addresses: []string{}, AutoDNS: true, DNS: []string{}}
}

func unique(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !slices.Contains(result, value) {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func run(ctx context.Context, name string, args ...string) (string, error) {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
