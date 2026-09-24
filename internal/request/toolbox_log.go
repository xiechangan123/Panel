package request

// ToolboxLogScan 日志扫描请求
type ToolboxLogScan struct {
	Type string `form:"type" json:"type" validate:"required && in:panel,website,mysql,docker,system"`
}

// ToolboxLogClean 日志清理请求
type ToolboxLogClean struct {
	Type  string   `form:"type" json:"type" validate:"required && in:panel,website,mysql,docker,system"`
	Paths []string `form:"paths" json:"paths" validate:"required"`
}
