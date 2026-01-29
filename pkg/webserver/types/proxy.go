package types

import "time"

// CacheConfig 缓存配置
type CacheConfig struct {
	// 缓存时长，状态码 -> 时长，如: {"200 302": "10m", "404": "1m", "any": "5m"}
	Valid map[string]string `form:"valid" json:"valid"`

	// 不缓存条件 (proxy_cache_bypass + proxy_no_cache)
	// 常用值: "$cookie_nocache", "$arg_nocache", "$http_pragma", "$http_authorization"
	NoCacheConditions []string `form:"no_cache_conditions" json:"no_cache_conditions"`

	// 过期缓存使用策略 (proxy_cache_use_stale)
	// 可选值: "error", "timeout", "updating", "http_500", "http_502", "http_503", "http_504"
	UseStale []string `form:"use_stale" json:"use_stale"`

	// 后台更新 (proxy_cache_background_update)
	BackgroundUpdate bool `form:"background_update" json:"background_update"`

	// 缓存锁 (proxy_cache_lock)，防止缓存击穿
	Lock bool `form:"lock" json:"lock"`

	// 最小请求次数 (proxy_cache_min_uses)，请求 N 次后才缓存
	MinUses int `form:"min_uses" json:"min_uses"`

	// 缓存方法 (proxy_cache_methods)，默认 GET HEAD
	Methods []string `form:"methods" json:"methods"`

	// 自定义缓存键 (proxy_cache_key)
	Key string `form:"key" json:"key"`
}

// Proxy 反向代理配置
type Proxy struct {
	Location        string            `form:"location" json:"location" validate:"required"` // 匹配路径，如: "/", "/api", "~ ^/api/v[0-9]+/"
	Pass            string            `form:"pass" json:"pass" validate:"required"`         // 代理地址，如: "http://example.com", "http://backend"
	Host            string            `form:"host" json:"host"`                             // 代理 Host，如: "example.com"
	SNI             string            `form:"sni" json:"sni"`                               // 代理 SNI，如: "example.com"
	Cache           *CacheConfig      `form:"cache" json:"cache"`                           // 缓存配置，nil 表示禁用缓存
	Buffering       bool              `form:"buffering" json:"buffering"`                   // 是否启用缓冲
	Resolver        []string          `form:"resolver" json:"resolver"`                     // 自定义 DNS 解析器配置，如: ["8.8.8.8", "ipv6=off"]
	ResolverTimeout time.Duration     `form:"resolver_timeout" json:"resolver_timeout"`     // DNS 解析超时时间，如: 5 * time.Second
	Headers         map[string]string `form:"headers" json:"headers"`                       // 自定义请求头，如: map["X-Custom-Header"] = "value"
	Replaces        map[string]string `form:"replaces" json:"replaces"`                     // 响应内容替换，如: map["/old"] = "/new"
}

// Upstream 上游服务器配置
type Upstream struct {
	Name            string            `form:"name" json:"name" validate:"required"`       // 上游名称，如: "backend"
	Servers         map[string]string `form:"servers" json:"servers" validate:"required"` // 上游服务器及配置，如: map["server1"] = "weight=5 resolve"
	Algo            string            `form:"algo" json:"algo"`                           // 负载均衡算法，如: "least_conn", "ip_hash"
	Keepalive       int               `form:"keepalive" json:"keepalive"`                 // 保持连接数，如: 32
	Resolver        []string          `form:"resolver" json:"resolver"`                   // 自定义 DNS 解析器配置，如: ["8.8.8.8", "ipv6=off"]
	ResolverTimeout time.Duration     `form:"resolver_timeout" json:"resolver_timeout"`   // DNS 解析超时时间，如: 5 * time.Second
}
