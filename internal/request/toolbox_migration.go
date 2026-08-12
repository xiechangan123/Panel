package request

// ToolboxMigrationConnection 迁移连接信息
type ToolboxMigrationConnection struct {
	SourcePanel string `json:"source_panel" validate:"in:,acepanel,baota,onepanel"`
	URL         string `json:"url" validate:"required"`
	TokenID     uint   `json:"token_id"`
	Token       string `json:"token"`
	APIKey      string `json:"api_key"`
}

// ToolboxMigrationItems 迁移选择项
type ToolboxMigrationItems struct {
	Items         []ToolboxMigrationSelectedItem `json:"items"`
	Websites      []ToolboxMigrationWebsite      `json:"websites"`
	Databases     []ToolboxMigrationDatabase     `json:"databases"`
	DatabaseUsers []ToolboxMigrationDatabaseUser `json:"database_users"`
	Projects      []ToolboxMigrationProject      `json:"projects"`
	StopOnMig     bool                           `json:"stop_on_mig"` // 兼容旧版 AcePanel 迁移请求

	SkipIncompatibleItems  bool `json:"skip_incompatible_items"`
	StopSourceDuringBackup bool `json:"stop_source_during_backup"`
}

// ToolboxMigrationSelectedItem 迁移资源选择项
type ToolboxMigrationSelectedItem struct {
	Key        string `json:"key" validate:"required"`
	TargetPath string `json:"target_path"`
	TargetUser string `json:"target_user"`
}

// ToolboxMigrationWebsite 迁移网站项
type ToolboxMigrationWebsite struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Path   string `json:"path"` // 网站目录
	Status bool   `json:"status"`
}

// ToolboxMigrationDatabase 迁移数据库项
type ToolboxMigrationDatabase struct {
	Type     string `json:"type" validate:"in:mysql,postgresql,clickhouse"` // mysql / postgresql / clickhouse
	Name     string `json:"name"`
	ServerID uint   `json:"server_id"`
	Server   string `json:"server"` // 服务器名称
}

// ToolboxMigrationDatabaseUser 迁移数据库用户项
type ToolboxMigrationDatabaseUser struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"` // 仅 mysql
	ServerID uint   `json:"server_id"`
	Server   string `json:"server"`                                         // 服务器名称
	Type     string `json:"type" validate:"in:mysql,postgresql,clickhouse"` // mysql / postgresql / clickhouse
}

// ToolboxMigrationProject 迁移项目项
type ToolboxMigrationProject struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`        // 来源项目目录
	TargetPath string `json:"target_path"` // 目标项目目录
	TargetUser string `json:"target_user"` // 目标运行用户，为空时沿用来源映射
}

// ToolboxMigrationExec 远程执行命令请求
type ToolboxMigrationExec struct {
	Command string `json:"command" validate:"required"`
}
