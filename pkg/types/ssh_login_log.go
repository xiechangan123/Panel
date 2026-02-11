package types

// SSHLoginLog SSH 登录日志条目
type SSHLoginLog struct {
	Time   string `json:"time"`
	User   string `json:"user"`
	IP     string `json:"ip"`
	Port   string `json:"port"`
	Method string `json:"method"`
	Status string `json:"status"`
}
