package types

// CronConfig 计划任务结构化配置
type CronConfig struct {
	Type    string   `json:"type"`    // 子类型：backup 时为 website/mysql/postgres；cutoff 时为 website/container
	Targets []string `json:"targets"` // 目标列表
	Storage uint     `json:"storage"` // 存储 ID（0=本地）
	Keep    uint     `json:"keep"`    // 保留份数
}
