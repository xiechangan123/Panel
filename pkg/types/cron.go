package types

// CronConfig 计划任务结构化配置
type CronConfig struct {
	Type    string   `json:"type"`    // 子类型：backup 时为 website/mysql/postgres；cutoff 时为 website/container
	Targets []string `json:"targets"` // 目标列表
	Storage uint     `json:"storage"` // 存储 ID（0=本地）
	Keep    uint     `json:"keep"`    // 保留份数
	// URL 任务专用
	URL      string            `json:"url"`
	Method   string            `json:"method"`   // GET/POST/PUT/DELETE/PATCH/HEAD
	Headers  map[string]string `json:"headers"`  // 自定义请求头
	Body     string            `json:"body"`     // 请求体
	Timeout  uint              `json:"timeout"`  // 超时时间（秒）
	Insecure bool              `json:"insecure"` // 忽略证书校验
	Retries  uint              `json:"retries"`  // 失败重试次数
}
