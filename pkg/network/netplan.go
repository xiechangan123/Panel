package network

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"
)

var netplanGroups = []string{"ethernets", "wifis", "bonds", "bridges", "vlans"}

type netplanMatch struct {
	group      string
	id         string
	definition map[string]any
}

func loadNetplan(ctx context.Context, items []Interface) error {
	config, err := getNetplan(ctx)
	if err != nil {
		return err
	}
	for i := range items {
		matches := findNetplan(config, items[i])
		if len(matches) == 0 {
			items[i].Reason = "no matching netplan interface definition"
			continue
		}
		if len(matches) > 1 {
			items[i].Reason = "multiple netplan interface definitions match this interface"
			continue
		}
		items[i].IPv4 = netplanFamily(matches[0].definition, true, items[i].CurrentIPv4)
		items[i].IPv6 = netplanFamily(matches[0].definition, false, items[i].CurrentIPv6)
		items[i].ConfiguredMTU = valueInt(matches[0].definition["mtu"])
		items[i].Editable = true
	}
	return nil
}

func updateNetplan(ctx context.Context, item Interface, config Config) error {
	document, err := getNetplan(ctx)
	if err != nil {
		return err
	}
	matches := findNetplan(document, item)
	if len(matches) != 1 {
		return &ValidationError{Message: "netplan interface definition no longer exists"}
	}
	match := matches[0]
	definition := match.definition
	definition["dhcp4"] = config.IPv4.Mode == ModeAuto
	definition["dhcp6"] = config.IPv6.Mode == ModeAuto
	definition["accept-ra"] = config.IPv6.Mode == ModeAuto
	definition["addresses"] = append(append([]string{}, config.IPv4.Addresses...), config.IPv6.Addresses...)
	definition["routes"] = netplanRoutes(definition["routes"], config)
	delete(definition, "gateway4")
	delete(definition, "gateway6")
	nameservers := valueMap(definition["nameservers"])
	nameservers["addresses"] = append(append([]string{}, config.IPv4.DNS...), config.IPv6.DNS...)
	definition["nameservers"] = nameservers
	for suffix, family := range map[string]FamilyConfig{"4": config.IPv4, "6": config.IPv6} {
		overrides := valueMap(definition["dhcp"+suffix+"-overrides"])
		overrides["use-dns"] = family.AutoDNS
		definition["dhcp"+suffix+"-overrides"] = overrides
	}
	if config.MTU == 0 {
		delete(definition, "mtu")
	} else {
		definition["mtu"] = config.MTU
	}

	key := match.group + "." + match.id
	data, err := yaml.Marshal(definition)
	if err != nil {
		return err
	}
	if _, err = run(ctx, "netplan", "set", "--origin-hint=99-acepanel", key+"="+strings.TrimSpace(string(data))); err != nil {
		return err
	}
	_ = os.Chmod("/etc/netplan/99-acepanel.yaml", 0600)
	if _, err = run(ctx, "netplan", "generate"); err != nil {
		return err
	}
	_, err = run(ctx, "netplan", "apply")
	return err
}

func getNetplan(ctx context.Context) (map[string]any, error) {
	output, err := run(ctx, "netplan", "get")
	if err != nil {
		return nil, err
	}
	config := make(map[string]any)
	if err = yaml.Unmarshal([]byte(output), &config); err != nil {
		return nil, err
	}
	return config, nil
}

func findNetplan(config map[string]any, item Interface) []netplanMatch {
	network := valueMap(config["network"])
	var result []netplanMatch
	for _, group := range netplanGroups {
		for id, raw := range valueMap(network[group]) {
			definition := valueMap(raw)
			match := valueMap(definition["match"])
			mac := valueString(match["macaddress"])
			matched := id == item.Name || valueString(definition["set-name"]) == item.Name ||
				(mac != "" && strings.EqualFold(mac, item.MAC))
			if pattern := valueString(match["name"]); !matched && pattern != "" {
				matched, _ = filepath.Match(pattern, item.Name)
			}
			if matched {
				result = append(result, netplanMatch{group: group, id: id, definition: definition})
			}
		}
	}
	return result
}

func netplanFamily(definition map[string]any, ipv4 bool, current []string) FamilyConfig {
	suffix := "4"
	if !ipv4 {
		suffix = "6"
	}
	addresses := netplanAddresses(definition["addresses"], ipv4)
	automatic := valueBool(definition["dhcp"+suffix])
	if !ipv4 {
		automatic = automatic || valueBool(definition["accept-ra"])
		if _, configured := definition["accept-ra"]; !configured && len(addresses) == 0 && len(current) > 0 {
			automatic = true
		}
	}
	mode := ModeDisabled
	if automatic {
		mode = ModeAuto
	} else if len(addresses) > 0 {
		mode = ModeManual
	}
	autoDNS := true
	if value, exists := valueMap(definition["dhcp"+suffix+"-overrides"])["use-dns"]; exists {
		autoDNS = valueBool(value)
	}
	return FamilyConfig{
		Mode: mode, Addresses: addresses, Gateway: netplanGateway(definition, ipv4),
		AutoDNS: autoDNS, DNS: netplanAddresses(valueMap(definition["nameservers"])["addresses"], ipv4),
	}
}

func netplanRoutes(raw any, config Config) []any {
	var result []any
	for _, current := range valueSlice(raw) {
		route := valueMap(current)
		to := valueString(route["to"])
		if to != "default" && to != "0.0.0.0/0" && to != "::/0" {
			result = append(result, current)
		}
	}
	if config.IPv4.Gateway != "" {
		result = append(result, map[string]any{"to": "default", "via": config.IPv4.Gateway})
	}
	if config.IPv6.Gateway != "" {
		result = append(result, map[string]any{"to": "default", "via": config.IPv6.Gateway})
	}
	return result
}

func netplanGateway(definition map[string]any, ipv4 bool) string {
	key := "gateway6"
	if ipv4 {
		key = "gateway4"
	}
	if gateway := valueString(definition[key]); gateway != "" {
		return gateway
	}
	for _, current := range valueSlice(definition["routes"]) {
		route := valueMap(current)
		to, gateway := valueString(route["to"]), valueString(route["via"])
		address, err := netip.ParseAddr(gateway)
		if (to == "default" || to == "0.0.0.0/0" || to == "::/0") && err == nil && address.Is4() == ipv4 {
			return gateway
		}
	}
	return ""
}

func netplanAddresses(raw any, ipv4 bool) []string {
	var result []string
	for _, current := range valueSlice(raw) {
		value := fmt.Sprint(current)
		if strings.Contains(value, "/") {
			if prefix, err := netip.ParsePrefix(value); err == nil && prefix.Addr().Is4() == ipv4 {
				result = append(result, value)
			}
		} else if address, err := netip.ParseAddr(value); err == nil && address.Is4() == ipv4 {
			result = append(result, value)
		}
	}
	return unique(result)
}

func valueMap(value any) map[string]any {
	result, _ := value.(map[string]any)
	if result == nil {
		result = make(map[string]any)
	}
	return result
}

func valueSlice(value any) []any {
	switch current := value.(type) {
	case []any:
		return current
	case []string:
		result := make([]any, len(current))
		for i := range current {
			result[i] = current[i]
		}
		return result
	default:
		return nil
	}
}

func valueString(value any) string {
	result, _ := value.(string)
	return result
}

func valueBool(value any) bool {
	result, _ := strconv.ParseBool(fmt.Sprint(value))
	return result
}

func valueInt(value any) int {
	result, _ := strconv.Atoi(fmt.Sprint(value))
	return result
}
