package request

type MonitorSetting struct {
	Enabled  bool `json:"enabled"`
	Days     uint `json:"days"`
	Interval uint `json:"interval" validate:"required && min:1 && max:120"` // 采集间隔（分钟），最小 1
}

type MonitorList struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}
