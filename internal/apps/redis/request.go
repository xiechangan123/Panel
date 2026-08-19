package redis

import "github.com/acepanel/panel/v3/pkg/types"

// ConfigTune Redis 协议兼容服务的配置调整
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

// ClientKill 踢除客户端连接请求
type ClientKill struct {
	ID int64 `form:"id" json:"id" validate:"required"`
}

// SlowLogEntry 慢日志条目
type SlowLogEntry struct {
	ID         int64  `json:"id"`
	Time       string `json:"time"`
	DurationUs int64  `json:"duration_us"`
	Command    string `json:"command"`
	Client     string `json:"client"`
}

// Client 客户端连接信息
type Client struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
	Name string `json:"name"`
	DB   string `json:"db"`
	Age  string `json:"age"`
	Idle string `json:"idle"`
	Cmd  string `json:"cmd"`
}

// Memory 内存诊断信息
type Memory struct {
	Doctor string     `json:"doctor"`
	Items  []types.NV `json:"items"`
}
