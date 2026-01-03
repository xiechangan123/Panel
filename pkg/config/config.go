package config

import (
	"os"

	"go.yaml.in/yaml/v4"

	"github.com/acepanel/panel/pkg/io"
)

const configPath = "/opt/ace/panel/storage/config.yml"

// Config 面板配置结构体
type Config struct {
	App      AppConfig      `yaml:"app"`
	HTTP     HTTPConfig     `yaml:"http"`
	Database DatabaseConfig `yaml:"database"`
	Session  SessionConfig  `yaml:"session"`
}

type AppConfig struct {
	Debug    bool   `yaml:"debug"`
	Key      string `yaml:"key"`
	Locale   string `yaml:"locale"`
	Timezone string `yaml:"timezone"`
	Root     string `yaml:"root"`
}

type HTTPConfig struct {
	Debug      bool     `yaml:"debug"`
	Port       uint     `yaml:"port"`
	Entrance   string   `yaml:"entrance"`
	TLS        bool     `yaml:"tls"`
	IPHeader   string   `yaml:"ip_header"`
	BindDomain []string `yaml:"bind_domain"`
	BindIP     []string `yaml:"bind_ip"`
	BindUA     []string `yaml:"bind_ua"`
}

type DatabaseConfig struct {
	Debug bool `yaml:"debug"`
}

type SessionConfig struct {
	Lifetime uint `yaml:"lifetime"`
}

func Load() (*Config, error) {
	path := configPath
	if !io.Exists(path) {
		path = "config.yml" // For testing purpose
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var conf Config
	decoder := yaml.NewDecoder(file)
	if err = decoder.Decode(&conf); err != nil {
		return nil, err
	}

	return &conf, nil
}

func Save(conf *Config) error {
	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func Check(conf *Config) (bool, error) {
	currentConf, err := Load()
	if err != nil {
		return false, err
	}

	currentData, err := yaml.Marshal(currentConf)
	if err != nil {
		return false, err
	}

	newData, err := yaml.Marshal(conf)
	if err != nil {
		return false, err
	}

	return string(currentData) == string(newData), nil
}
