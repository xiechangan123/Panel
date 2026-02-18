package request

// WebsiteStatSetting 网站统计设置
type WebsiteStatSetting struct {
	Days uint `json:"days" validate:"required|min:1|max:365"`
}
