package request

// WebsiteDefaultSite 默认站点设置,ID 为 0 表示面板内置默认页
type WebsiteDefaultSite struct {
	ID uint `json:"id" form:"id"`
}
