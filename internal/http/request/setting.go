package request

import "net/http"

type PanelSetting struct {
	Name        string   `json:"name" validate:"required"`
	Channel     string   `json:"channel" validate:"required|in:stable,beta"`
	Locale      string   `json:"locale" validate:"required"`
	Entrance    string   `json:"entrance" validate:"required"`
	OfflineMode bool     `json:"offline_mode"`
	AutoUpdate  bool     `json:"auto_update"`
	TwoFA       bool     `json:"two_fa"`
	Lifetime    uint     `json:"lifetime" validate:"required|min:10|max:43200"` // 登录超时，单位：分
	BindDomain  []string `json:"bind_domain"`
	BindIP      []string `json:"bind_ip"`
	BindUA      []string `json:"bind_ua"`
	WebsitePath string   `json:"website_path" validate:"required"`
	BackupPath  string   `json:"backup_path" validate:"required"`
	Username    string   `json:"username" validate:"required"`
	Password    string   `json:"password" validate:"password"`
	Email       string   `json:"email" validate:"required"`
	Port        uint     `json:"port" validate:"required|min:1|max:65535"`
	HTTPS       bool     `json:"https"`
	Cert        string   `json:"cert" validate:"required"`
	Key         string   `json:"key" validate:"required"`
}

func (r *PanelSetting) Rules(_ *http.Request) map[string]string {
	return map[string]string{
		"BindDomain.*": "required",
		"BindIP.*":     "required|ip",
		"BindUA.*":     "required",
	}
}
