package clickhouse

// UpdateConfig 更新配置
type UpdateConfig struct {
	Config string `form:"config" json:"config" validate:"required"`
}

// SetDefaultPassword 设置 default 用户密码
type SetDefaultPassword struct {
	Password string `form:"password" json:"password" validate:"required|password"`
}

// ConfigTune ClickHouse 配置调整
type ConfigTune struct {
	// 网络
	ListenHost string `form:"listen_host" json:"listen_host"`
	HTTPPort   string `form:"http_port" json:"http_port"`
	TCPPort    string `form:"tcp_port" json:"tcp_port"`
	// 性能
	MaxMemoryUsage string `form:"max_memory_usage" json:"max_memory_usage"`
	MaxThreads     string `form:"max_threads" json:"max_threads"`
	// 路径
	Path    string `form:"path" json:"path"`
	TmpPath string `form:"tmp_path" json:"tmp_path"`
	// 日志
	LogLevel string `form:"log_level" json:"log_level"`
}
