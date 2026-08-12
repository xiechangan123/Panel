package network

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

func loadNetworkManager(ctx context.Context, items []Interface) error {
	for i := range items {
		uuid, err := nmDeviceValue(ctx, items[i].Name, "GENERAL.CON-UUID")
		if err != nil || uuid == "" || uuid == "--" {
			items[i].Reason = "no active NetworkManager connection profile"
			continue
		}
		ipv4, err4 := nmFamily(ctx, uuid, "ipv4")
		ipv6, err6 := nmFamily(ctx, uuid, "ipv6")
		mtu, errMTU := nmConnectionValue(ctx, uuid, "mtu")
		if err4 != nil || err6 != nil || errMTU != nil {
			items[i].Reason = "failed to read NetworkManager connection profile"
			continue
		}
		items[i].IPv4 = ipv4
		items[i].IPv6 = ipv6
		items[i].ConfiguredMTU, _ = strconv.Atoi(mtu)
		items[i].Editable = true
	}
	return nil
}

func updateNetworkManager(ctx context.Context, config Config) error {
	uuid, err := nmDeviceValue(ctx, config.Name, "GENERAL.CON-UUID")
	if err != nil || uuid == "" || uuid == "--" {
		return &ValidationError{Message: "no active NetworkManager connection profile"}
	}
	args := []string{"connection", "modify", "uuid", uuid}
	args = append(args, nmFamilyArgs("ipv4", config.IPv4)...)
	args = append(args, nmFamilyArgs("ipv6", config.IPv6)...)
	args = append(args, "mtu", strconv.Itoa(config.MTU))
	if _, err = run(ctx, "nmcli", args...); err != nil {
		return err
	}
	if _, err = run(ctx, "nmcli", "connection", "verify", "uuid", uuid); err != nil {
		return err
	}
	if _, err = run(ctx, "nmcli", "device", "reapply", config.Name); err == nil {
		return nil
	}
	_, err = run(ctx, "nmcli", "connection", "up", "uuid", uuid, "ifname", config.Name)
	return err
}

func nmFamily(ctx context.Context, uuid, name string) (FamilyConfig, error) {
	values := make(map[string]string)
	for _, field := range []string{"method", "addresses", "gateway", "dns", "ignore-auto-dns"} {
		value, err := nmConnectionValue(ctx, uuid, name+"."+field)
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
		Addresses: nmList(values["addresses"]),
		Gateway:   nmEmpty(values["gateway"]),
		AutoDNS:   values["ignore-auto-dns"] != "yes",
		DNS:       nmList(values["dns"]),
	}, nil
}

func nmFamilyArgs(name string, config FamilyConfig) []string {
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

func nmConnectionValue(ctx context.Context, uuid, field string) (string, error) {
	return nmValue(ctx, "-g", field, "connection", "show", "uuid", uuid)
}

func nmDeviceValue(ctx context.Context, name, field string) (string, error) {
	return nmValue(ctx, "-g", field, "device", "show", name)
}

func nmValue(ctx context.Context, args ...string) (string, error) {
	value, err := run(ctx, "nmcli", args...)
	if err != nil {
		return "", fmt.Errorf("failed to read NetworkManager configuration: %w", err)
	}
	return strings.ReplaceAll(strings.TrimSpace(value), `\:`, ":"), nil
}

func nmEmpty(value string) string {
	if value == "--" {
		return ""
	}
	return value
}

func nmList(value string) []string {
	if value == "" || value == "--" {
		return []string{}
	}
	return unique(strings.FieldsFunc(value, func(r rune) bool { return r == '\n' || r == ',' }))
}
