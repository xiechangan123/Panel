package rsync

type Module struct {
	Name       string `form:"name" json:"name" validate:"required && regex:\"^[a-zA-Z0-9_.-]+$\""`
	Path       string `json:"path" validate:"required && unix_path"`
	Comment    string `json:"comment"`
	ReadOnly   bool   `json:"read_only"`
	AuthUser   string `json:"auth_user" validate:"required"`
	Secret     string `json:"secret" validate:"required"`
	HostsAllow string `json:"hosts_allow"`
}
