package network

import (
	"testing"

	"github.com/libtnb/assert/check"
	"github.com/libtnb/assert/must"
)

func TestValidate(t *testing.T) {
	valid := Config{
		Name: "eth0",
		MTU:  1500,
		IPv4: FamilyConfig{Mode: ModeManual, Addresses: []string{"192.0.2.10/24"}, Gateway: "198.51.100.1", DNS: []string{"1.1.1.1"}},
		IPv6: FamilyConfig{Mode: ModeDisabled},
	}
	tests := []struct {
		name   string
		change func(*Config)
	}{
		{name: "invalid interface", change: func(config *Config) { config.Name = "../eth0" }},
		{name: "invalid CIDR", change: func(config *Config) { config.IPv4.Addresses = []string{"192.0.2.10"} }},
		{name: "wrong family", change: func(config *Config) { config.IPv4.Gateway = "2001:db8::1" }},
		{name: "duplicate DNS", change: func(config *Config) { config.IPv4.DNS = []string{"1.1.1.1", "1.1.1.1"} }},
		{name: "manual without address", change: func(config *Config) { config.IPv4.Addresses = nil }},
		{name: "both disabled", change: func(config *Config) { config.IPv4 = FamilyConfig{Mode: ModeDisabled} }},
		{name: "invalid MTU", change: func(config *Config) { config.MTU = 67 }},
		{name: "IPv4 network address", change: func(config *Config) { config.IPv4.Addresses = []string{"192.0.2.0/24"} }},
		{name: "IPv6 subnet-router anycast", change: func(config *Config) {
			config.IPv6 = FamilyConfig{Mode: ModeManual, Addresses: []string{"2001:db8:0:b4::/64"}}
		}},
	}
	must.NoError(t, Validate(valid))
	pointToPoint := valid
	pointToPoint.IPv4.Addresses = []string{"192.0.2.0/31", "198.51.100.7/32"}
	pointToPoint.IPv6 = FamilyConfig{Mode: ModeManual, Addresses: []string{"2001:db8::/127", "2001:db8:1::/128"}}
	must.NoError(t, Validate(pointToPoint))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.change(&config)
			check.ErrorIs(t, Validate(config), ErrValidation)
		})
	}
}

func TestCheckApplied(t *testing.T) {
	config := Config{
		Name: "eth0",
		IPv4: FamilyConfig{Mode: ModeManual, Addresses: []string{"192.0.2.10/24"}},
		IPv6: FamilyConfig{Mode: ModeManual, Addresses: []string{"2001:DB8:0:00b4::10/64"}},
	}
	link := func(flags, ipv6 string) string {
		return `[{"ifname":"eth0","flags":[` + flags + `],"addr_info":[` +
			`{"family":"inet","local":"192.0.2.10","prefixlen":24},` +
			`{"family":"inet6","local":"2001:db8:0:b4::10","prefixlen":64` + ipv6 + `}]}]`
	}

	must.NoError(t, checkApplied(config, link(`"BROADCAST","UP","LOWER_UP"`, "")))
	check.ErrorContains(t, checkApplied(config, link(`"BROADCAST"`, "")), "is down")
	check.ErrorContains(t, checkApplied(config, link(`"UP"`, `,"tentative":true`)), "did not pass duplicate address detection")
	check.ErrorIs(t, checkApplied(config, link(`"UP"`, `,"tentative":true,"dadfailed":true`)), ErrValidation)

	missing := config
	missing.IPv6.Addresses = []string{"2001:db8:0:b4::20/64"}
	check.ErrorContains(t, checkApplied(missing, link(`"UP"`, "")), "was not applied")
	check.Error(t, checkApplied(config, `Device "eth0" does not exist.`))
}

func TestIfupdownParse(t *testing.T) {
	backend := &ifupdownBackend{}
	file := backend.parse("/etc/network/interfaces", `auto eth0
iface eth0 inet dhcp
    dns-nameservers 1.1.1.1
    mtu 1500

iface eth0 inet6 static
    address 2001:db8::10/64
`)
	must.Len(t, file.stanzas, 2)
	check.DeepEqual(t, backend.family(file.stanzas, "inet"), FamilyConfig{
		Mode:      ModeAuto,
		Addresses: []string{},
		AutoDNS:   true,
		DNS:       []string{"1.1.1.1"},
	})
	check.DeepEqual(t, backend.family(file.stanzas, "inet6"), FamilyConfig{
		Mode:      ModeManual,
		Addresses: []string{"2001:db8::10/64"},
		AutoDNS:   true,
		DNS:       []string{},
	})
}
