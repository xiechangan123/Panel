package types

// PanelConfig 面板配置结构体
type PanelConfig struct {
	App      PanelAppConfig      `yaml:"app"`
	HTTP     PanelHTTPConfig     `yaml:"http"`
	Database PanelDatabaseConfig `yaml:"database"`
	Session  PanelSessionConfig  `yaml:"session"`
}

type PanelAppConfig struct {
	Debug    bool   `yaml:"debug"`
	Key      string `yaml:"key"`
	Locale   string `yaml:"locale"`
	Timezone string `yaml:"timezone"`
	Root     string `yaml:"root"`
}

type PanelHTTPConfig struct {
	Debug      bool     `yaml:"debug"`
	Port       uint     `yaml:"port"`
	Entrance   string   `yaml:"entrance"`
	TLS        bool     `yaml:"tls"`
	BindDomain []string `yaml:"bind_domain"`
	BindIP     []string `yaml:"bind_ip"`
	BindUA     []string `yaml:"bind_ua"`
}

type PanelDatabaseConfig struct {
	Debug bool `yaml:"debug"`
}

type PanelSessionConfig struct {
	Lifetime uint `yaml:"lifetime"`
}
