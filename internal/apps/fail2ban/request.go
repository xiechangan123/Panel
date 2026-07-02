package fail2ban

type Add struct {
	Name        string `json:"name" validate:"required"`
	Type        string `json:"type" validate:"required && in:service,website"`
	MaxRetry    int    `json:"maxretry" validate:"required && min:1"`
	FindTime    int    `json:"findtime" validate:"required && min:1"`
	BanTime     int    `json:"bantime" validate:"required && min:1"`
	WebsiteName string `json:"website_name" validate:"required_if:Type,website"`
	WebsiteMode string `json:"website_mode" validate:"required_if:Type,website && in:cc,path"`
	WebsitePath string `json:"website_path"`
}

type Delete struct {
	Name string `json:"name" validate:"required"`
}

type BanList struct {
	Name string `json:"name" validate:"required"`
}

type Unban struct {
	Name string `json:"name" validate:"required"`
	IP   string `json:"ip" validate:"required && ip"`
}

type SetWhiteList struct {
	IP string `json:"ip" validate:"required"`
}
