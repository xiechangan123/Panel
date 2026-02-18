package websitestat

// LogEntry 解析后的访问日志条目
type LogEntry struct {
	Site        string // 来自 syslog tag
	URI         string // 请求 URI
	Status      int    // HTTP 状态码
	Bytes       uint64 // 响应体大小
	UA          string // User-Agent
	IP          string // 客户端 IP
	Method      string // 请求方法
	Body        string // 请求体（仅 400-599 状态码时保留）
	ContentType string // 响应 Content-Type（PV 判定用）
	ReqLength   uint64 // 请求大小（入站流量）
}

// HourSnapshot 小时粒度快照
type HourSnapshot struct {
	PV        uint64 `json:"pv"`
	UV        uint64 `json:"uv"`
	IP        uint64 `json:"ip"`
	Bandwidth uint64 `json:"bandwidth"`
	Requests  uint64 `json:"requests"`
	Errors    uint64 `json:"errors"`
	Spiders   uint64 `json:"spiders"`
}

// SiteSnapshot 站点快照（用于 DB flush）
type SiteSnapshot struct {
	PV        uint64            `json:"pv"`
	UV        uint64            `json:"uv"`
	IP        uint64            `json:"ip"`
	Bandwidth uint64            `json:"bandwidth"`
	Requests  uint64            `json:"requests"`
	Errors    uint64            `json:"errors"`
	Spiders   uint64            `json:"spiders"`
	Hours     [24]*HourSnapshot `json:"-"`
}

// RealtimeStats 实时统计
type RealtimeStats struct {
	Bandwidth float64 `json:"bandwidth"` // 字节/秒
	RPS       float64 `json:"rps"`       // 请求/秒
}

// ErrorEntry 错误日志条目
type ErrorEntry struct {
	Site   string
	URI    string
	Method string
	IP     string
	UA     string
	Body   string
	Status int
}

// SiteDetailSnapshot 站点详细统计快照（用于 DrainDetailStats）
type SiteDetailSnapshot struct {
	Spiders map[string]uint64       // spider_name → requests
	Clients map[string]*ClientCount // "browser|os" → count
	IPs     map[string]*IPCount     // ip → count
	URIs    map[string]*URICount    // uri → count
}

// ClientCount 客户端计数
type ClientCount struct {
	Requests uint64
}

// IPCount IP 计数
type IPCount struct {
	Requests  uint64
	Bandwidth uint64
}

// URICount URI 计数
type URICount struct {
	Requests  uint64
	Bandwidth uint64
	Errors    uint64
}
