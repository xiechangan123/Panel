package types

import "time"

// Proxy 反向代理配置
type Proxy struct {
	Location        string            `form:"location" json:"location" validate:"required"` // 匹配路径，如: "/", "/api", "~ ^/api/v[0-9]+/"
	AutoRefresh     bool              `form:"auto_refresh" json:"auto_refresh"`             // 是否自动刷新解析
	Pass            string            `form:"pass" json:"pass" validate:"required"`         // 代理地址，如: "http://example.com", "http://backend"
	Host            string            `form:"host" json:"host"`                             // 代理 Host，如: "example.com"
	SNI             string            `form:"sni" json:"sni"`                               // 代理 SNI，如: "example.com"
	Cache           bool              `form:"cache" json:"cache"`                           // 是否启用缓存
	Buffering       bool              `form:"buffering" json:"buffering"`                   // 是否启用缓冲
	Resolver        []string          `form:"resolver" json:"resolver"`                     // 自定义 DNS 解析器配置，如: ["8.8.8.8", "ipv6=off"]
	ResolverTimeout time.Duration     `form:"resolver_timeout" json:"resolver_timeout"`     // DNS 解析超时时间，如: 5 * time.Second
	Replaces        map[string]string `form:"replaces" json:"replaces"`                     // 响应内容替换，如: map["/old"] = "/new"
}

// Upstream 上游服务器配置
type Upstream struct {
	Servers   map[string]string `form:"servers" json:"servers" validate:"required"` // 上游服务器及权重，如: map["server1"] = "weight=5"
	Algo      string            `form:"algo" json:"algo"`                           // 负载均衡算法，如: "least_conn", "ip_hash"
	Keepalive int               `form:"keepalive" json:"keepalive"`                 // 保持连接数，如: 32
}
