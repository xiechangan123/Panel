package frp

import "regexp"

var (
	userCaptureRegex  = regexp.MustCompile(`(?m)^User=(.*)$`)
	groupCaptureRegex = regexp.MustCompile(`(?m)^Group=(.*)$`)
	userRegex         = regexp.MustCompile(`(?m)^User=.*$`)
	groupRegex        = regexp.MustCompile(`(?m)^Group=.*$`)
	serviceRegex      = regexp.MustCompile(`(?m)^\[Service\]$`)
)

type UserInfo struct {
	User  string `json:"user"`
	Group string `json:"group"`
}

// ConfD conf.d 目录下的配置文件，只承载代理与访问者
type ConfD struct {
	Proxies  []Proxy   `toml:"proxies,omitempty"`
	Visitors []Visitor `toml:"visitors,omitempty"`
}

// Proxy 代理配置，字段对齐 frp pkg/config/v1 的 ProxyConfigurer 实现
type Proxy struct {
	Name    string `toml:"name" form:"name" json:"name" validate:"required && regex:\"^[a-zA-Z0-9_.-]+$\""`
	Type    string `toml:"type" json:"type" validate:"required && in:tcp,udp,http,https,tcpmux,stcp,sudp,xtcp"`
	Enabled *bool  `toml:"enabled,omitempty" json:"enabled"`

	LocalIP   string `toml:"localIP,omitempty" json:"local_ip"`
	LocalPort int    `toml:"localPort,omitempty" json:"local_port"`

	// tcp、udp
	RemotePort int `toml:"remotePort,omitempty" json:"remote_port"`

	// http、https、tcpmux
	CustomDomains []string `toml:"customDomains,omitempty" json:"custom_domains"`
	Subdomain     string   `toml:"subdomain,omitempty" json:"subdomain"`

	// http、tcpmux
	HTTPUser        string `toml:"httpUser,omitempty" json:"http_user"`
	HTTPPassword    string `toml:"httpPassword,omitempty" json:"http_password"`
	RouteByHTTPUser string `toml:"routeByHTTPUser,omitempty" json:"route_by_http_user"`

	// http
	Locations         []string          `toml:"locations,omitempty" json:"locations"`
	HostHeaderRewrite string            `toml:"hostHeaderRewrite,omitempty" json:"host_header_rewrite"`
	RequestHeaders    *HeaderOperations `toml:"requestHeaders,omitempty" json:"request_headers"`
	ResponseHeaders   *HeaderOperations `toml:"responseHeaders,omitempty" json:"response_headers"`

	// tcpmux
	Multiplexer string `toml:"multiplexer,omitempty" json:"multiplexer"`

	// stcp、sudp、xtcp
	SecretKey  string   `toml:"secretKey,omitempty" json:"secret_key"`
	AllowUsers []string `toml:"allowUsers,omitempty" json:"allow_users"`

	Transport    *ProxyTransport `toml:"transport,omitempty" json:"transport"`
	LoadBalancer *LoadBalancer   `toml:"loadBalancer,omitempty" json:"load_balancer"`
	HealthCheck  *HealthCheck    `toml:"healthCheck,omitempty" json:"health_check"`
	Plugin       *Plugin         `toml:"plugin,omitempty" json:"plugin"`

	Metadatas   map[string]string `toml:"metadatas,omitempty" json:"metadatas"`
	Annotations map[string]string `toml:"annotations,omitempty" json:"annotations"`
}

// Visitor 访问者配置，对应 stcp、sudp、xtcp 的客户端一侧
type Visitor struct {
	Name    string `toml:"name" form:"name" json:"name" validate:"required && regex:\"^[a-zA-Z0-9_.-]+$\""`
	Type    string `toml:"type" json:"type" validate:"required && in:stcp,sudp,xtcp"`
	Enabled *bool  `toml:"enabled,omitempty" json:"enabled"`

	ServerUser string `toml:"serverUser,omitempty" json:"server_user"`
	ServerName string `toml:"serverName,omitempty" json:"server_name"`
	SecretKey  string `toml:"secretKey,omitempty" json:"secret_key"`
	BindAddr   string `toml:"bindAddr,omitempty" json:"bind_addr"`
	BindPort   int    `toml:"bindPort,omitempty" json:"bind_port"`

	// xtcp
	Protocol          string `toml:"protocol,omitempty" json:"protocol"`
	KeepTunnelOpen    bool   `toml:"keepTunnelOpen,omitempty" json:"keep_tunnel_open"`
	MaxRetriesAnHour  int    `toml:"maxRetriesAnHour,omitempty" json:"max_retries_an_hour"`
	MinRetryInterval  int    `toml:"minRetryInterval,omitempty" json:"min_retry_interval"`
	FallbackTo        string `toml:"fallbackTo,omitempty" json:"fallback_to"`
	FallbackTimeoutMs int    `toml:"fallbackTimeoutMs,omitempty" json:"fallback_timeout_ms"`

	Transport *VisitorTransport `toml:"transport,omitempty" json:"transport"`
}

type ProxyTransport struct {
	UseEncryption        bool   `toml:"useEncryption,omitempty" json:"use_encryption"`
	UseCompression       bool   `toml:"useCompression,omitempty" json:"use_compression"`
	BandwidthLimit       string `toml:"bandwidthLimit,omitempty" json:"bandwidth_limit"`
	BandwidthLimitMode   string `toml:"bandwidthLimitMode,omitempty" json:"bandwidth_limit_mode"`
	ProxyProtocolVersion string `toml:"proxyProtocolVersion,omitempty" json:"proxy_protocol_version"`
}

type VisitorTransport struct {
	UseEncryption  bool `toml:"useEncryption,omitempty" json:"use_encryption"`
	UseCompression bool `toml:"useCompression,omitempty" json:"use_compression"`
}

type LoadBalancer struct {
	Group    string `toml:"group,omitempty" json:"group"`
	GroupKey string `toml:"groupKey,omitempty" json:"group_key"`
}

type HealthCheck struct {
	Type            string `toml:"type" json:"type"`
	TimeoutSeconds  int    `toml:"timeoutSeconds,omitempty" json:"timeout_seconds"`
	MaxFailed       int    `toml:"maxFailed,omitempty" json:"max_failed"`
	IntervalSeconds int    `toml:"intervalSeconds,omitempty" json:"interval_seconds"`
	Path            string `toml:"path,omitempty" json:"path"`
}

type HeaderOperations struct {
	Set map[string]string `toml:"set,omitempty" json:"set"`
}

// Plugin 客户端插件配置，各插件字段的并集
type Plugin struct {
	Type string `toml:"type" json:"type"`

	// unix_domain_socket
	UnixPath string `toml:"unixPath,omitempty" json:"unix_path"`

	// http_proxy、static_file
	HTTPUser     string `toml:"httpUser,omitempty" json:"http_user"`
	HTTPPassword string `toml:"httpPassword,omitempty" json:"http_password"`

	// socks5
	Username string `toml:"username,omitempty" json:"username"`
	Password string `toml:"password,omitempty" json:"password"`

	// static_file
	LocalPath   string `toml:"localPath,omitempty" json:"local_path"`
	StripPrefix string `toml:"stripPrefix,omitempty" json:"strip_prefix"`

	// https2http、https2https、http2https、http2http、tls2raw
	LocalAddr         string `toml:"localAddr,omitempty" json:"local_addr"`
	HostHeaderRewrite string `toml:"hostHeaderRewrite,omitempty" json:"host_header_rewrite"`
	CrtPath           string `toml:"crtPath,omitempty" json:"crt_path"`
	KeyPath           string `toml:"keyPath,omitempty" json:"key_path"`
}

// ServerTune frps 可视化参数
type ServerTune struct {
	BindAddr               string `form:"bind_addr" json:"bind_addr"`
	BindPort               string `form:"bind_port" json:"bind_port"`
	KCPBindPort            string `form:"kcp_bind_port" json:"kcp_bind_port"`
	QUICBindPort           string `form:"quic_bind_port" json:"quic_bind_port"`
	ProxyBindAddr          string `form:"proxy_bind_addr" json:"proxy_bind_addr"`
	SubDomainHost          string `form:"sub_domain_host" json:"sub_domain_host"`
	MaxPortsPerClient      string `form:"max_ports_per_client" json:"max_ports_per_client"`
	UDPPacketSize          string `form:"udp_packet_size" json:"udp_packet_size"`
	DetailedErrorsToClient string `form:"detailed_errors_to_client" json:"detailed_errors_to_client"`

	AuthMethod string `form:"auth_method" json:"auth_method"`
	AuthToken  string `form:"auth_token" json:"auth_token"`

	VhostHTTPPort         string `form:"vhost_http_port" json:"vhost_http_port"`
	VhostHTTPSPort        string `form:"vhost_https_port" json:"vhost_https_port"`
	VhostHTTPTimeout      string `form:"vhost_http_timeout" json:"vhost_http_timeout"`
	TCPMuxHTTPConnectPort string `form:"tcpmux_http_connect_port" json:"tcpmux_http_connect_port"`
	Custom404Page         string `form:"custom_404_page" json:"custom_404_page"`

	WebServerAddr        string `form:"web_server_addr" json:"web_server_addr"`
	WebServerPort        string `form:"web_server_port" json:"web_server_port"`
	WebServerUser        string `form:"web_server_user" json:"web_server_user"`
	WebServerPassword    string `form:"web_server_password" json:"web_server_password"`
	WebServerPprofEnable string `form:"web_server_pprof_enable" json:"web_server_pprof_enable"`
	EnablePrometheus     string `form:"enable_prometheus" json:"enable_prometheus"`

	TransportMaxPoolCount            string `form:"transport_max_pool_count" json:"transport_max_pool_count"`
	TransportTCPMux                  string `form:"transport_tcp_mux" json:"transport_tcp_mux"`
	TransportTCPMuxKeepaliveInterval string `form:"transport_tcp_mux_keepalive_interval" json:"transport_tcp_mux_keepalive_interval"`
	TransportTCPKeepalive            string `form:"transport_tcp_keepalive" json:"transport_tcp_keepalive"`
	TransportHeartbeatTimeout        string `form:"transport_heartbeat_timeout" json:"transport_heartbeat_timeout"`
	TransportTLSForce                string `form:"transport_tls_force" json:"transport_tls_force"`
	TransportTLSCertFile             string `form:"transport_tls_cert_file" json:"transport_tls_cert_file"`
	TransportTLSKeyFile              string `form:"transport_tls_key_file" json:"transport_tls_key_file"`
	TransportTLSTrustedCaFile        string `form:"transport_tls_trusted_ca_file" json:"transport_tls_trusted_ca_file"`

	LogTo                string `form:"log_to" json:"log_to"`
	LogLevel             string `form:"log_level" json:"log_level"`
	LogMaxDays           string `form:"log_max_days" json:"log_max_days"`
	LogDisablePrintColor string `form:"log_disable_print_color" json:"log_disable_print_color"`
}

// ClientTune frpc 可视化公共参数
type ClientTune struct {
	User              string `form:"user" json:"user"`
	ServerAddr        string `form:"server_addr" json:"server_addr"`
	ServerPort        string `form:"server_port" json:"server_port"`
	LoginFailExit     string `form:"login_fail_exit" json:"login_fail_exit"`
	NatHoleStunServer string `form:"nat_hole_stun_server" json:"nat_hole_stun_server"`
	DNSServer         string `form:"dns_server" json:"dns_server"`
	UDPPacketSize     string `form:"udp_packet_size" json:"udp_packet_size"`

	AuthMethod string `form:"auth_method" json:"auth_method"`
	AuthToken  string `form:"auth_token" json:"auth_token"`

	TransportProtocol                string `form:"transport_protocol" json:"transport_protocol"`
	TransportPoolCount               string `form:"transport_pool_count" json:"transport_pool_count"`
	TransportTCPMux                  string `form:"transport_tcp_mux" json:"transport_tcp_mux"`
	TransportTCPMuxKeepaliveInterval string `form:"transport_tcp_mux_keepalive_interval" json:"transport_tcp_mux_keepalive_interval"`
	TransportDialServerTimeout       string `form:"transport_dial_server_timeout" json:"transport_dial_server_timeout"`
	TransportDialServerKeepalive     string `form:"transport_dial_server_keepalive" json:"transport_dial_server_keepalive"`
	TransportHeartbeatInterval       string `form:"transport_heartbeat_interval" json:"transport_heartbeat_interval"`
	TransportHeartbeatTimeout        string `form:"transport_heartbeat_timeout" json:"transport_heartbeat_timeout"`
	TransportConnectServerLocalIP    string `form:"transport_connect_server_local_ip" json:"transport_connect_server_local_ip"`
	TransportProxyURL                string `form:"transport_proxy_url" json:"transport_proxy_url"`
	TransportTLSEnable               string `form:"transport_tls_enable" json:"transport_tls_enable"`
	TransportTLSCertFile             string `form:"transport_tls_cert_file" json:"transport_tls_cert_file"`
	TransportTLSKeyFile              string `form:"transport_tls_key_file" json:"transport_tls_key_file"`
	TransportTLSTrustedCaFile        string `form:"transport_tls_trusted_ca_file" json:"transport_tls_trusted_ca_file"`
	TransportTLSServerName           string `form:"transport_tls_server_name" json:"transport_tls_server_name"`

	WebServerAddr        string `form:"web_server_addr" json:"web_server_addr"`
	WebServerPort        string `form:"web_server_port" json:"web_server_port"`
	WebServerUser        string `form:"web_server_user" json:"web_server_user"`
	WebServerPassword    string `form:"web_server_password" json:"web_server_password"`
	WebServerPprofEnable string `form:"web_server_pprof_enable" json:"web_server_pprof_enable"`

	LogTo                string `form:"log_to" json:"log_to"`
	LogLevel             string `form:"log_level" json:"log_level"`
	LogMaxDays           string `form:"log_max_days" json:"log_max_days"`
	LogDisablePrintColor string `form:"log_disable_print_color" json:"log_disable_print_color"`
}
