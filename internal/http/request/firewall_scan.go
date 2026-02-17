package request

// FirewallScanSetting 扫描感知设置
type FirewallScanSetting struct {
	Enabled    bool     `json:"enabled"`
	Days       uint     `json:"days" validate:"min:1|max:365"`
	Interfaces []string `json:"interfaces"`
}
