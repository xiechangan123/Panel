package frp

import (
	"github.com/spf13/cast"

	"github.com/acepanel/panel/v3/internal/apps/confval"
)

// tuneField 可视化参数与 TOML 键的对应关系，kind 决定写入时的字面量类型
type tuneField struct {
	key   string
	value *string
	kind  byte // s 字符串、i 整数、b 布尔
}

// literal 把表单里的字符串还原成 TOML 字面量，空串表示注释掉该项
func (f tuneField) literal() any {
	if *f.value == "" {
		return ""
	}

	switch f.kind {
	case 'i':
		return cast.ToInt(*f.value)
	case 'b':
		return cast.ToBool(*f.value)
	default:
		return *f.value
	}
}

// readTune 从配置内容读出各参数
func readTune(config string, fields []tuneField) {
	for _, f := range fields {
		*f.value = confval.GetTOML(config, f.key)
	}
}

// writeTune 把各参数写回配置内容
func writeTune(config string, fields []tuneField) string {
	for _, f := range fields {
		config = confval.SetTOML(config, f.key, f.literal())
	}

	return config
}

func serverFields(t *ServerTune) []tuneField {
	return []tuneField{
		{"bindAddr", &t.BindAddr, 's'},
		{"bindPort", &t.BindPort, 'i'},
		{"kcpBindPort", &t.KCPBindPort, 'i'},
		{"quicBindPort", &t.QUICBindPort, 'i'},
		{"proxyBindAddr", &t.ProxyBindAddr, 's'},
		{"subDomainHost", &t.SubDomainHost, 's'},
		{"maxPortsPerClient", &t.MaxPortsPerClient, 'i'},
		{"udpPacketSize", &t.UDPPacketSize, 'i'},
		{"detailedErrorsToClient", &t.DetailedErrorsToClient, 'b'},

		{"auth.method", &t.AuthMethod, 's'},
		{"auth.token", &t.AuthToken, 's'},

		{"vhostHTTPPort", &t.VhostHTTPPort, 'i'},
		{"vhostHTTPSPort", &t.VhostHTTPSPort, 'i'},
		{"vhostHTTPTimeout", &t.VhostHTTPTimeout, 'i'},
		{"tcpmuxHTTPConnectPort", &t.TCPMuxHTTPConnectPort, 'i'},
		{"custom404Page", &t.Custom404Page, 's'},

		{"webServer.addr", &t.WebServerAddr, 's'},
		{"webServer.port", &t.WebServerPort, 'i'},
		{"webServer.user", &t.WebServerUser, 's'},
		{"webServer.password", &t.WebServerPassword, 's'},
		{"webServer.pprofEnable", &t.WebServerPprofEnable, 'b'},
		{"enablePrometheus", &t.EnablePrometheus, 'b'},

		{"transport.maxPoolCount", &t.TransportMaxPoolCount, 'i'},
		{"transport.tcpMux", &t.TransportTCPMux, 'b'},
		{"transport.tcpMuxKeepaliveInterval", &t.TransportTCPMuxKeepaliveInterval, 'i'},
		{"transport.tcpKeepalive", &t.TransportTCPKeepalive, 'i'},
		{"transport.heartbeatTimeout", &t.TransportHeartbeatTimeout, 'i'},
		{"transport.tls.force", &t.TransportTLSForce, 'b'},
		{"transport.tls.certFile", &t.TransportTLSCertFile, 's'},
		{"transport.tls.keyFile", &t.TransportTLSKeyFile, 's'},
		{"transport.tls.trustedCaFile", &t.TransportTLSTrustedCaFile, 's'},

		{"log.to", &t.LogTo, 's'},
		{"log.level", &t.LogLevel, 's'},
		{"log.maxDays", &t.LogMaxDays, 'i'},
		{"log.disablePrintColor", &t.LogDisablePrintColor, 'b'},
	}
}

func clientFields(t *ClientTune) []tuneField {
	return []tuneField{
		{"user", &t.User, 's'},
		{"serverAddr", &t.ServerAddr, 's'},
		{"serverPort", &t.ServerPort, 'i'},
		{"loginFailExit", &t.LoginFailExit, 'b'},
		{"natHoleStunServer", &t.NatHoleStunServer, 's'},
		{"dnsServer", &t.DNSServer, 's'},
		{"udpPacketSize", &t.UDPPacketSize, 'i'},

		{"auth.method", &t.AuthMethod, 's'},
		{"auth.token", &t.AuthToken, 's'},

		{"transport.protocol", &t.TransportProtocol, 's'},
		{"transport.poolCount", &t.TransportPoolCount, 'i'},
		{"transport.tcpMux", &t.TransportTCPMux, 'b'},
		{"transport.tcpMuxKeepaliveInterval", &t.TransportTCPMuxKeepaliveInterval, 'i'},
		{"transport.dialServerTimeout", &t.TransportDialServerTimeout, 'i'},
		{"transport.dialServerKeepalive", &t.TransportDialServerKeepalive, 'i'},
		{"transport.heartbeatInterval", &t.TransportHeartbeatInterval, 'i'},
		{"transport.heartbeatTimeout", &t.TransportHeartbeatTimeout, 'i'},
		{"transport.connectServerLocalIP", &t.TransportConnectServerLocalIP, 's'},
		{"transport.proxyURL", &t.TransportProxyURL, 's'},
		{"transport.tls.enable", &t.TransportTLSEnable, 'b'},
		{"transport.tls.certFile", &t.TransportTLSCertFile, 's'},
		{"transport.tls.keyFile", &t.TransportTLSKeyFile, 's'},
		{"transport.tls.trustedCaFile", &t.TransportTLSTrustedCaFile, 's'},
		{"transport.tls.serverName", &t.TransportTLSServerName, 's'},

		{"webServer.addr", &t.WebServerAddr, 's'},
		{"webServer.port", &t.WebServerPort, 'i'},
		{"webServer.user", &t.WebServerUser, 's'},
		{"webServer.password", &t.WebServerPassword, 's'},
		{"webServer.pprofEnable", &t.WebServerPprofEnable, 'b'},

		{"log.to", &t.LogTo, 's'},
		{"log.level", &t.LogLevel, 's'},
		{"log.maxDays", &t.LogMaxDays, 'i'},
		{"log.disablePrintColor", &t.LogDisablePrintColor, 'b'},
	}
}
