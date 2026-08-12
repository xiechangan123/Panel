package network

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"
)

var ifacePattern = regexp.MustCompile(`^\s*iface\s+(\S+)\s+(inet6?)\s+(\S+)`)

type interfacesFile struct {
	path    string
	lines   []string
	stanzas []interfacesStanza
}

type interfacesStanza struct {
	name, family, method string
	start, end           int
	options              []string
}

func loadIfupdown(items []Interface) error {
	files, err := readInterfacesFiles()
	if err != nil {
		return err
	}
	for i := range items {
		file, stanzas, reason := findIfupdown(files, items[i].Name)
		if reason != "" {
			items[i].Reason = reason
			continue
		}
		items[i].IPv4 = ifupdownFamily(stanzas, "inet")
		items[i].IPv6 = ifupdownFamily(stanzas, "inet6")
		for _, stanza := range stanzas {
			if mtu := stanzaValue(stanza, "mtu"); mtu != "" {
				items[i].ConfiguredMTU, _ = strconv.Atoi(mtu)
				break
			}
		}
		items[i].Editable = file != nil
	}
	return nil
}

func updateIfupdown(ctx context.Context, config Config) error {
	files, err := readInterfacesFiles()
	if err != nil {
		return err
	}
	file, stanzas, reason := findIfupdown(files, config.Name)
	if reason != "" || file == nil {
		return &ValidationError{Message: reason}
	}
	start, end := stanzas[0].start, stanzas[len(stanzas)-1].end
	unknown4, unknown6 := unknownOptions(stanzas, "inet"), unknownOptions(stanzas, "inet6")
	lines := append([]string{}, file.lines[:start]...)
	lines = append(lines, renderIfupdown(config, unknown4, unknown6)...)
	lines = append(lines, file.lines[end:]...)

	tmp, err := os.CreateTemp(filepath.Dir(file.path), ".acepanel-interfaces-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err = tmp.WriteString(strings.Join(lines, "")); err == nil {
		err = tmp.Chmod(0644)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if _, err = run(ctx, "ifquery", "--interfaces", tmp.Name(), config.Name); err != nil {
		return err
	}
	if err = os.Rename(tmp.Name(), file.path); err != nil {
		return err
	}
	if hasCommand("ifreload") {
		_, err = run(ctx, "ifreload", "-c")
		return err
	}
	if _, err = run(ctx, "ifdown", "--force", config.Name); err != nil {
		return err
	}
	_, err = run(ctx, "ifup", config.Name)
	return err
}

func readInterfacesFiles() ([]interfacesFile, error) {
	paths := []string{"/etc/network/interfaces"}
	included, _ := filepath.Glob("/etc/network/interfaces.d/*")
	paths = append(paths, included...)
	files := make([]interfacesFile, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		files = append(files, parseInterfaces(path, string(data)))
	}
	return files, nil
}

func parseInterfaces(path, content string) interfacesFile {
	lines := strings.SplitAfter(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	file := interfacesFile{path: path, lines: lines}
	for i := 0; i < len(lines); {
		match := ifacePattern.FindStringSubmatch(lines[i])
		if len(match) == 0 {
			i++
			continue
		}
		end := i + 1
		for end < len(lines) {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") && lines[end][0] != ' ' && lines[end][0] != '\t' {
				break
			}
			end++
		}
		file.stanzas = append(file.stanzas, interfacesStanza{
			name: match[1], family: match[2], method: match[3], start: i, end: end,
			options: append([]string{}, lines[i+1:end]...),
		})
		i = end
	}
	return file
}

func findIfupdown(files []interfacesFile, name string) (*interfacesFile, []interfacesStanza, string) {
	var file *interfacesFile
	var stanzas []interfacesStanza
	for i := range files {
		for _, line := range files[i].lines {
			fields := strings.Fields(line)
			if len(fields) > 1 && fields[0] == "mapping" {
				return nil, nil, "ifupdown mapping configuration cannot be edited safely"
			}
		}
		for _, stanza := range files[i].stanzas {
			if stanza.name != name {
				continue
			}
			if file != nil && file.path != files[i].path {
				return nil, nil, "ifupdown interface is defined in multiple files"
			}
			if stanzaValue(stanza, "inherits") != "" {
				return nil, nil, "ifupdown inheritance cannot be edited safely"
			}
			file = &files[i]
			stanzas = append(stanzas, stanza)
		}
	}
	if len(stanzas) == 0 {
		return nil, nil, "no matching ifupdown interface definition"
	}
	for _, stanza := range file.stanzas {
		if stanza.start > stanzas[0].start && stanza.start < stanzas[len(stanzas)-1].end && stanza.name != name {
			return nil, nil, "ifupdown interface has interleaved definitions"
		}
	}
	return file, stanzas, ""
}

func ifupdownFamily(stanzas []interfacesStanza, family string) FamilyConfig {
	config := emptyFamily()
	for _, stanza := range stanzas {
		if stanza.family != family {
			continue
		}
		if stanza.method == "dhcp" || stanza.method == "auto" {
			config.Mode = ModeAuto
		} else if stanza.method == "static" && config.Mode != ModeAuto {
			config.Mode = ModeManual
		}
		if address := stanzaAddress(stanza); address != "" {
			config.Addresses = append(config.Addresses, address)
		}
		if config.Gateway == "" {
			config.Gateway = stanzaValue(stanza, "gateway")
		}
		config.DNS = append(config.DNS, strings.Fields(stanzaValue(stanza, "dns-nameservers"))...)
		if family == "inet6" && config.Mode == ModeAuto {
			config.AutoDNS = stanzaValue(stanza, "dhcp") != "0"
		}
	}
	config.Addresses, config.DNS = unique(config.Addresses), unique(config.DNS)
	return config
}

func renderIfupdown(config Config, unknown4, unknown6 []string) []string {
	lines := renderIfupdownFamily(config.Name, "inet", config.IPv4, config.MTU, unknown4)
	lines = append(lines, "\n")
	return append(lines, renderIfupdownFamily(config.Name, "inet6", config.IPv6, config.MTU, unknown6)...)
}

func renderIfupdownFamily(name, family string, config FamilyConfig, mtu int, unknown []string) []string {
	method := lo.Switch[string, string](config.Mode).
		Case(ModeAuto, lo.If(family == "inet", "dhcp").Else("auto")).
		Case(ModeManual, "static").
		Default("manual")
	lines := []string{fmt.Sprintf("iface %s %s %s\n", name, family, method)}
	addresses := config.Addresses
	if config.Mode == ModeManual && len(addresses) > 0 {
		lines = append(lines, "    address "+addresses[0]+"\n")
		addresses = addresses[1:]
	}
	if config.Gateway != "" {
		lines = append(lines, "    gateway "+config.Gateway+"\n")
	}
	if len(config.DNS) > 0 {
		lines = append(lines, "    dns-nameservers "+strings.Join(config.DNS, " ")+"\n")
	}
	if family == "inet6" && config.Mode == ModeAuto {
		dhcp := 0
		if config.AutoDNS {
			dhcp = 1
		}
		lines = append(lines, "    accept_ra 2\n", fmt.Sprintf("    dhcp %d\n", dhcp))
	} else if family == "inet6" && config.Mode == ModeDisabled {
		lines = append(lines, "    accept_ra 0\n", "    autoconf 0\n")
	}
	if mtu > 0 {
		lines = append(lines, fmt.Sprintf("    mtu %d\n", mtu))
	}
	lines = append(lines, unknown...)
	for _, address := range addresses {
		lines = append(lines, "\n", fmt.Sprintf("iface %s %s static\n", name, family), "    address "+address+"\n")
	}
	return lines
}

func unknownOptions(stanzas []interfacesStanza, family string) []string {
	managed := []string{"address", "netmask", "gateway", "dns-nameservers", "mtu", "accept_ra", "autoconf", "dhcp"}
	var result []string
	for _, stanza := range stanzas {
		if stanza.family != family {
			continue
		}
		for _, line := range stanza.options {
			fields := strings.Fields(line)
			if len(fields) > 0 && !slices.Contains(managed, fields[0]) {
				result = append(result, line)
			}
		}
	}
	return result
}

func stanzaValue(stanza interfacesStanza, key string) string {
	for _, line := range stanza.options {
		fields := strings.Fields(line)
		if len(fields) > 1 && fields[0] == key {
			return strings.Join(fields[1:], " ")
		}
	}
	return ""
}

func stanzaAddress(stanza interfacesStanza) string {
	address, mask := stanzaValue(stanza, "address"), stanzaValue(stanza, "netmask")
	if address == "" || mask == "" || strings.Contains(address, "/") {
		return address
	}
	if bits, err := strconv.Atoi(mask); err == nil {
		return address + "/" + strconv.Itoa(bits)
	}
	if parsed := net.ParseIP(mask); parsed != nil {
		if bits, _ := net.IPMask(parsed.To4()).Size(); bits > 0 {
			return address + "/" + strconv.Itoa(bits)
		}
	}
	return address
}
