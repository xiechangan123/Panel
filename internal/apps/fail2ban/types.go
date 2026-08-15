package fail2ban

type Jail struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	MaxRetry int    `json:"max_retry"`
	FindTime int    `json:"find_time"`
	BanTime  int    `json:"ban_time"`

	// 以下几项在新增时由面板推导，之后随规则文件原样保留
	Filter  string `json:"filter"`
	Port    string `json:"port"`
	LogPath string `json:"log_path"`
}
