package fail2ban

type Add struct {
	Name        string `json:"name" validate:"required && regex:\"^[a-zA-Z0-9_.-]+$\""`
	Type        string `json:"type" validate:"required && in:service,website"`
	MaxRetry    int    `json:"maxretry" validate:"required && min:1"`
	FindTime    int    `json:"findtime" validate:"required && min:1"`
	BanTime     int    `json:"bantime" validate:"required && min:1"`
	WebsiteName string `json:"website_name" validate:"required_if:Type,website"`
	WebsiteMode string `json:"website_mode" validate:"required_if:Type,website && in:cc,path"`
	WebsitePath string `json:"website_path"`
}

type JailName struct {
	Name string `form:"name" json:"name" validate:"required && regex:\"^[a-zA-Z0-9_.-]+$\""`
}

// Update 只允许改这几项，过滤器、端口与日志路径随规则文件原样保留
type Update struct {
	Name     string `form:"name" json:"name" validate:"required && regex:\"^[a-zA-Z0-9_.-]+$\""`
	Enabled  bool   `json:"enabled"`
	MaxRetry int    `json:"max_retry" validate:"required && min:1"`
	FindTime int    `json:"find_time" validate:"required && min:1"`
	BanTime  int    `json:"ban_time" validate:"required && min:1"`
}

type Unban struct {
	Name string `form:"name" json:"name" validate:"required"`
	IP   string `json:"ip" validate:"required && ip"`
}

type SetWhiteList struct {
	IP string `json:"ip" validate:"required"`
}
