package frp

import (
	"github.com/spf13/cast"

	"github.com/acepanel/panel/v3/internal/apps/confval"
)

type tuneKind uint8

const (
	tuneString tuneKind = iota
	tuneInt
	tuneBool
)

// tuneField 可视化参数与 TOML 键的对应关系，表单一律用字符串收值，kind 决定写回时的字面量类型
type tuneField struct {
	key   string
	value *string
	kind  tuneKind
}

// literal 空串表示注释掉该项
func (f tuneField) literal() any {
	if *f.value == "" {
		return ""
	}

	switch f.kind {
	case tuneInt:
		return cast.ToInt(*f.value)
	case tuneBool:
		return cast.ToBool(*f.value)
	default:
		return *f.value
	}
}

func readTune(config string, fields []tuneField) {
	for _, f := range fields {
		*f.value = confval.GetTOML(config, f.key)
	}
}

func writeTune(config string, fields []tuneField) string {
	for _, f := range fields {
		config = confval.SetTOML(config, f.key, f.literal())
	}

	return config
}

func serverFields(t *ServerTune) []tuneField {
	return []tuneField{
		{"bindAddr", &t.BindAddr, tuneString},
		{"bindPort", &t.BindPort, tuneInt},
		{"kcpBindPort", &t.KCPBindPort, tuneInt},
		{"quicBindPort", &t.QUICBindPort, tuneInt},
		{"proxyBindAddr", &t.ProxyBindAddr, tuneString},
		{"subDomainHost", &t.SubDomainHost, tuneString},
		{"maxPortsPerClient", &t.MaxPortsPerClient, tuneInt},
		{"udpPacketSize", &t.UDPPacketSize, tuneInt},
		{"detailedErrorsToClient", &t.DetailedErrorsToClient, tuneBool},

		{"auth.method", &t.AuthMethod, tuneString},
		{"auth.token", &t.AuthToken, tuneString},

		{"vhostHTTPPort", &t.VhostHTTPPort, tuneInt},
		{"vhostHTTPSPort", &t.VhostHTTPSPort, tuneInt},
		{"vhostHTTPTimeout", &t.VhostHTTPTimeout, tuneInt},
		{"tcpmuxHTTPConnectPort", &t.TCPMuxHTTPConnectPort, tuneInt},
		{"custom404Page", &t.Custom404Page, tuneString},

		{"webServer.addr", &t.WebServerAddr, tuneString},
		{"webServer.port", &t.WebServerPort, tuneInt},
		{"webServer.user", &t.WebServerUser, tuneString},
		{"webServer.password", &t.WebServerPassword, tuneString},
		{"webServer.pprofEnable", &t.WebServerPprofEnable, tuneBool},
		{"enablePrometheus", &t.EnablePrometheus, tuneBool},

		{"transport.maxPoolCount", &t.TransportMaxPoolCount, tuneInt},
		{"transport.tcpMux", &t.TransportTCPMux, tuneBool},
		{"transport.tcpMuxKeepaliveInterval", &t.TransportTCPMuxKeepaliveInterval, tuneInt},
		{"transport.tcpKeepalive", &t.TransportTCPKeepalive, tuneInt},
		{"transport.heartbeatTimeout", &t.TransportHeartbeatTimeout, tuneInt},
		{"transport.tls.force", &t.TransportTLSForce, tuneBool},
		{"transport.tls.certFile", &t.TransportTLSCertFile, tuneString},
		{"transport.tls.keyFile", &t.TransportTLSKeyFile, tuneString},
		{"transport.tls.trustedCaFile", &t.TransportTLSTrustedCaFile, tuneString},

		{"log.to", &t.LogTo, tuneString},
		{"log.level", &t.LogLevel, tuneString},
		{"log.maxDays", &t.LogMaxDays, tuneInt},
		{"log.disablePrintColor", &t.LogDisablePrintColor, tuneBool},
	}
}

func clientFields(t *ClientTune) []tuneField {
	return []tuneField{
		{"user", &t.User, tuneString},
		{"serverAddr", &t.ServerAddr, tuneString},
		{"serverPort", &t.ServerPort, tuneInt},
		{"loginFailExit", &t.LoginFailExit, tuneBool},
		{"natHoleStunServer", &t.NatHoleStunServer, tuneString},
		{"dnsServer", &t.DNSServer, tuneString},
		{"udpPacketSize", &t.UDPPacketSize, tuneInt},

		{"auth.method", &t.AuthMethod, tuneString},
		{"auth.token", &t.AuthToken, tuneString},

		{"transport.protocol", &t.TransportProtocol, tuneString},
		{"transport.poolCount", &t.TransportPoolCount, tuneInt},
		{"transport.tcpMux", &t.TransportTCPMux, tuneBool},
		{"transport.tcpMuxKeepaliveInterval", &t.TransportTCPMuxKeepaliveInterval, tuneInt},
		{"transport.dialServerTimeout", &t.TransportDialServerTimeout, tuneInt},
		{"transport.dialServerKeepalive", &t.TransportDialServerKeepalive, tuneInt},
		{"transport.heartbeatInterval", &t.TransportHeartbeatInterval, tuneInt},
		{"transport.heartbeatTimeout", &t.TransportHeartbeatTimeout, tuneInt},
		{"transport.connectServerLocalIP", &t.TransportConnectServerLocalIP, tuneString},
		{"transport.proxyURL", &t.TransportProxyURL, tuneString},
		{"transport.tls.enable", &t.TransportTLSEnable, tuneBool},
		{"transport.tls.certFile", &t.TransportTLSCertFile, tuneString},
		{"transport.tls.keyFile", &t.TransportTLSKeyFile, tuneString},
		{"transport.tls.trustedCaFile", &t.TransportTLSTrustedCaFile, tuneString},
		{"transport.tls.serverName", &t.TransportTLSServerName, tuneString},

		{"webServer.addr", &t.WebServerAddr, tuneString},
		{"webServer.port", &t.WebServerPort, tuneInt},
		{"webServer.user", &t.WebServerUser, tuneString},
		{"webServer.password", &t.WebServerPassword, tuneString},
		{"webServer.pprofEnable", &t.WebServerPprofEnable, tuneBool},

		{"log.to", &t.LogTo, tuneString},
		{"log.level", &t.LogLevel, tuneString},
		{"log.maxDays", &t.LogMaxDays, tuneInt},
		{"log.disablePrintColor", &t.LogDisablePrintColor, tuneBool},
	}
}
