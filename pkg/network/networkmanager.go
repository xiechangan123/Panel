package network

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type networkManagerBackend struct{}

func (b *networkManagerBackend) Name() string { return "networkmanager" }

func (b *networkManagerBackend) available(ctx context.Context) bool {
	if !hasCommand("nmcli") {
		return false
	}
	output, err := run(ctx, "nmcli", "-t", "-f", "RUNNING", "general", "status")
	return err == nil && strings.EqualFold(output, "running")
}

func (b *networkManagerBackend) Load(ctx context.Context, items []Interface) error {
	for i := range items {
		uuid, err := b.uuid(ctx, items[i].Name)
		if err != nil {
			items[i].Reason = "no active NetworkManager connection profile"
			continue
		}
		ipv4, err4 := b.family(ctx, uuid, "ipv4")
		ipv6, err6 := b.family(ctx, uuid, "ipv6")
		mtu, errMTU := b.value(ctx, "-g", mtuProperty(items[i].Type), "connection", "show", "uuid", uuid)
		if err4 != nil || err6 != nil || errMTU != nil {
			items[i].Reason = "failed to read NetworkManager connection profile"
			continue
		}
		items[i].IPv4, items[i].IPv6 = ipv4, ipv6
		// 未设置 MTU 时返回 auto，解析失败即视为未配置
		items[i].ConfiguredMTU, _ = strconv.Atoi(mtu)
		items[i].Editable = true
	}
	return nil
}

func (b *networkManagerBackend) Apply(ctx context.Context, item Interface, config Config) (func(context.Context) error, error) {
	uuid, err := b.uuid(ctx, config.Name)
	if err != nil {
		return nil, fmt.Errorf("%w: no active NetworkManager connection profile", ErrValidation)
	}

	// 连接配置由 nmcli 直接改写，改回读取到的旧值即可撤销
	previous := Config{Name: item.Name, MTU: item.ConfiguredMTU, IPv4: item.IPv4, IPv6: item.IPv6}
	rollback := func(ctx context.Context) error { return b.modify(ctx, uuid, item.Type, previous) }
	if err = b.modify(ctx, uuid, item.Type, config); err != nil {
		return nil, err
	}
	return rollback, nil
}

func (b *networkManagerBackend) modify(ctx context.Context, uuid, kind string, config Config) error {
	mtu := "auto"
	if config.MTU > 0 {
		mtu = strconv.Itoa(config.MTU)
	}
	args := []string{"connection", "modify", "uuid", uuid}
	args = append(args, b.familyArgs("ipv4", config.IPv4)...)
	args = append(args, b.familyArgs("ipv6", config.IPv6)...)
	args = append(args, mtuProperty(kind), mtu)
	if _, err := run(ctx, "nmcli", args...); err != nil {
		return err
	}
	if _, err := run(ctx, "nmcli", "connection", "verify", "uuid", uuid); err != nil {
		return err
	}

	// reapply 可在不断链的情况下生效，不支持时回退到重新激活连接
	if _, err := run(ctx, "nmcli", "device", "reapply", config.Name); err == nil {
		return nil
	}
	_, err := run(ctx, "nmcli", "connection", "up", "uuid", uuid, "ifname", config.Name)
	return err
}

func (b *networkManagerBackend) uuid(ctx context.Context, name string) (string, error) {
	uuid, err := b.value(ctx, "-g", "GENERAL.CON-UUID", "device", "show", name)
	if err != nil {
		return "", err
	}
	if uuid == "" || uuid == "--" {
		return "", fmt.Errorf("interface %s has no connection profile", name)
	}
	return uuid, nil
}

func (b *networkManagerBackend) family(ctx context.Context, uuid, name string) (FamilyConfig, error) {
	values := make(map[string]string)
	for _, field := range []string{"method", "addresses", "gateway", "dns", "ignore-auto-dns"} {
		value, err := b.value(ctx, "-g", name+"."+field, "connection", "show", "uuid", uuid)
		if err != nil {
			return FamilyConfig{}, err
		}
		values[field] = value
	}

	mode := ModeDisabled
	switch values["method"] {
	case "auto", "dhcp":
		mode = ModeAuto
	case "manual":
		mode = ModeManual
	}
	return FamilyConfig{
		Mode:      mode,
		Addresses: b.list(values["addresses"]),
		Gateway:   b.empty(values["gateway"]),
		AutoDNS:   values["ignore-auto-dns"] != "yes",
		DNS:       b.list(values["dns"]),
	}, nil
}

func (b *networkManagerBackend) familyArgs(name string, config FamilyConfig) []string {
	ignoreAutoDNS := "yes"
	if config.AutoDNS {
		ignoreAutoDNS = "no"
	}
	return []string{
		name + ".method", config.Mode,
		name + ".addresses", strings.Join(config.Addresses, ","),
		name + ".gateway", config.Gateway,
		name + ".dns", strings.Join(config.DNS, ","),
		name + ".ignore-auto-dns", ignoreAutoDNS,
	}
}

func (b *networkManagerBackend) value(ctx context.Context, args ...string) (string, error) {
	value, err := run(ctx, "nmcli", args...)
	if err != nil {
		return "", fmt.Errorf("failed to read NetworkManager configuration: %w", err)
	}
	// nmcli 的输出会转义值内的冒号
	return strings.ReplaceAll(strings.TrimSpace(value), `\:`, ":"), nil
}

func (b *networkManagerBackend) empty(value string) string {
	if value == "--" {
		return ""
	}
	return value
}

func (b *networkManagerBackend) list(value string) []string {
	if value == "" || value == "--" {
		return []string{}
	}
	return unique(strings.FieldsFunc(value, func(r rune) bool { return r == '\n' || r == ',' }))
}

// mtuProperty MTU 属于具体链路层设置，nmcli 只接受带设置名的完整属性
func mtuProperty(kind string) string {
	if kind == "wifi" {
		return "802-11-wireless.mtu"
	}
	return "802-3-ethernet.mtu"
}
