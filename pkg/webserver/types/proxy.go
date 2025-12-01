package types

import "time"

// Proxy 反向代理配置
type Proxy struct {
	AutoRefresh     bool              // 是否自动刷新解析
	Pass            string            // 代理地址，如: "http://example.com", "http://backend"
	Host            string            // 代理 Host，如: "example.com"
	SNI             string            // 代理 SNI，如: "example.com"
	Cache           bool              // 是否启用缓存
	Buffering       bool              // 是否启用缓冲
	Resolver        []string          // 自定义 DNS 解析器配置，如: ["8.8.8.8", "ipv6=off"]
	ResolverTimeout time.Duration     // DNS 解析超时时间，如: 5 * time.Second
	Replaces        map[string]string // 响应内容替换，如: map["/old"] = "/new"
}

// Upstream 上游服务器配置
type Upstream struct {
	Name      string            // 上游名称，如: "backend"
	Servers   map[string]string // 上游服务器及权重，如: map["server1"] = "weight=5"
	Algo      string            // 负载均衡算法，如: "least_conn", "ip_hash"
	Keepalive int               // 保持连接数，如: 32
}
