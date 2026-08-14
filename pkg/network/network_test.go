package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	}
	require.NoError(t, Validate(valid))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.change(&config)
			assert.Error(t, Validate(config))
		})
	}
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
	require.Len(t, file.stanzas, 2)
	assert.Equal(t, ModeAuto, backend.family(file.stanzas, "inet").Mode)
	assert.Equal(t, []string{"2001:db8::10/64"}, backend.family(file.stanzas, "inet6").Addresses)
}
