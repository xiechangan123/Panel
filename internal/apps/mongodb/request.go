package mongodb

// UpdateConfig 更新配置
type UpdateConfig struct {
	Config string `form:"config" json:"config" validate:"required"`
}

// SetAdminPassword 设置 admin 密码
type SetAdminPassword struct {
	Password string `form:"password" json:"password" validate:"required|password"`
}

// ConfigTune MongoDB 配置调整
type ConfigTune struct {
	// 存储
	DbPath      string `form:"db_path" json:"db_path"`
	CacheSizeGB string `form:"cache_size_gb" json:"cache_size_gb"`
	// 网络
	Port   string `form:"port" json:"port"`
	BindIp string `form:"bind_ip" json:"bind_ip"`
	// 日志
	SystemLogPath string `form:"system_log_path" json:"system_log_path"`
	// 安全
	Authorization string `form:"authorization" json:"authorization"`
}
