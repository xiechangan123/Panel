package request

// ToolboxMigrationConnection 迁移来源连接信息
type ToolboxMigrationConnection struct {
	SourcePanel string `json:"source_panel" validate:"in:,acepanel,baota,onepanel"`
	URL         string `json:"url" validate:"required"`
	TokenID     uint   `json:"token_id"` // acepanel 专用
	Token       string `json:"token"`    // acepanel 专用
	APIKey      string `json:"api_key"`  // baota / onepanel 专用
}

// ToolboxMigrationStart 开始迁移请求
type ToolboxMigrationStart struct {
	Items       []ToolboxMigrationItem `json:"items" validate:"required"`
	SkipBlocked bool                   `json:"skip_blocked"` // 跳过不兼容项而非中止
	StopSource  bool                   `json:"stop_source"`  // 备份期间停止来源资源
}

// ToolboxMigrationItem 选中的迁移项
type ToolboxMigrationItem struct {
	Key        string `json:"key" validate:"required"`
	TargetPath string `json:"target_path"`
	TargetUser string `json:"target_user"` // 项目运行用户
}

// ToolboxMigrationExec 远程执行命令请求（供来源面板推送迁移时调用）
type ToolboxMigrationExec struct {
	Command string `json:"command" validate:"required"`
}
