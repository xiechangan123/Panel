package docker

type UpdateConfig struct {
	Config string `form:"config" json:"config" validate:"required"`
}

// LogOpts 日志配置选项
type LogOpts struct {
	MaxSize string `json:"max-size,omitempty"` // 日志文件最大大小，如 "10m"
	MaxFile string `json:"max-file,omitempty"` // 保存的日志文件份数，如 "3"
}

// Settings Docker daemon 设置
type Settings struct {
	RegistryMirrors    []string `json:"registry-mirrors,omitempty"`    // 注册表镜像
	InsecureRegistries []string `json:"insecure-registries,omitempty"` // 非安全镜像仓库
	LiveRestore        bool     `json:"live-restore,omitempty"`        // Live restore
	LogDriver          string   `json:"log-driver,omitempty"`          // 日志驱动
	LogOpts            LogOpts  `json:"log-opts"`                      // 日志配置选项
	CgroupDriver       string   `json:"cgroup-driver,omitempty"`       // cgroup 驱动（从 exec-opts 中提取）
	Hosts              []string `json:"hosts,omitempty"`               // Socket 路径
	DataRoot           string   `json:"data-root,omitempty"`           // 数据目录
	StorageDriver      string   `json:"storage-driver,omitempty"`      // 存储驱动
	DNS                []string `json:"dns,omitempty"`                 // DNS 配置
	FirewallBackend    string   `json:"firewall-backend,omitempty"`    // 防火墙后端 (iptables/nftables)
	Iptables           *bool    `json:"iptables,omitempty"`            // iptables 规则
	Ip6tables          *bool    `json:"ip6tables,omitempty"`           // ip6tables 规则
	IpForward          *bool    `json:"ip-forward,omitempty"`          // IP 转发
	IPv6               *bool    `json:"ipv6,omitempty"`                // IPv6 支持
	Bip                string   `json:"bip,omitempty"`                 // 默认 bridge 网络 IP 段
}

// UpdateSettings 更新设置请求
type UpdateSettings struct {
	Settings Settings `json:"settings" validate:"required"`
}
