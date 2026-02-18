package request

// WebsiteStatSetting 网站统计设置
type WebsiteStatSetting struct {
	Days          uint `json:"days" validate:"required|min:1|max:365"`
	ErrBufMax     int  `json:"err_buf_max" validate:"min:0|max:1000000"`
	UVMaxKeys     int  `json:"uv_max_keys" validate:"min:0|max:100000000"`
	IPMaxKeys     int  `json:"ip_max_keys" validate:"min:0|max:100000000"`
	DetailMaxKeys int  `json:"detail_max_keys" validate:"min:0|max:100000000"`
}
