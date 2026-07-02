package redis

type UpdateConfig struct {
	Config string `form:"config" json:"config" validate:"required"`
}

// ConfigTune Redis 配置调整
type ConfigTune struct {
	// 常规设置
	Bind         string `form:"bind" json:"bind"`
	Port         string `form:"port" json:"port" validate:"number && min:1 && max:65535"`
	Databases    string `form:"databases" json:"databases" validate:"number && min:1"`
	Requirepass  string `form:"requirepass" json:"requirepass"`
	Timeout      string `form:"timeout" json:"timeout" validate:"number"`
	TCPKeepalive string `form:"tcp_keepalive" json:"tcp_keepalive" validate:"number"`
	// 内存
	Maxmemory       string `form:"maxmemory" json:"maxmemory"`
	MaxmemoryPolicy string `form:"maxmemory_policy" json:"maxmemory_policy" validate:"in:noeviction,allkeys-lru,allkeys-lfu,allkeys-random,volatile-lru,volatile-lfu,volatile-random,volatile-ttl"`
	// 持久化
	Appendonly  string `form:"appendonly" json:"appendonly" validate:"in:yes,no"`
	Appendfsync string `form:"appendfsync" json:"appendfsync" validate:"in:always,everysec,no"`
}
