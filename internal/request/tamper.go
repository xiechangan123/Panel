package request

// TamperSetting 防篡改全局设置
type TamperSetting struct {
	Enabled       bool   `json:"enabled" form:"enabled"`
	Mode          string `json:"mode" form:"mode" validate:"omitempty && in:chattr,ebpf"`
	BlockNewFiles bool   `json:"block_new_files" form:"block_new_files"`
	LogDays       uint   `json:"log_days" form:"log_days"`
}

// TamperRuleCreate 新增保护规则
type TamperRuleCreate struct {
	Name     string   `json:"name" form:"name" validate:"required"`
	Path     string   `json:"path" form:"path" validate:"required"`
	Exts     []string `json:"exts" form:"exts"`
	Excludes []string `json:"excludes" form:"excludes"`
	Enabled  bool     `json:"enabled" form:"enabled"`
}

// TamperRuleUpdate 更新保护规则
type TamperRuleUpdate struct {
	ID       uint     `json:"id" form:"id" uri:"id" validate:"required"`
	Path     string   `json:"path" form:"path" validate:"required"`
	Exts     []string `json:"exts" form:"exts"`
	Excludes []string `json:"excludes" form:"excludes"`
	Enabled  bool     `json:"enabled" form:"enabled"`
}
