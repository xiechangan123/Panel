package webserver

import (
	"fmt"

	"github.com/acepanel/panel/pkg/webserver/apache"
	"github.com/acepanel/panel/pkg/webserver/nginx"
	"github.com/acepanel/panel/pkg/webserver/types"
)

// NewStaticVhost 创建纯静态虚拟主机实例
func NewStaticVhost(serverType Type, configDir string) (types.StaticVhost, error) {
	switch serverType {
	case TypeNginx:
		return nginx.NewStaticVhost(configDir)
	case TypeApache:
		return apache.NewStaticVhost(configDir)
	default:
		return nil, fmt.Errorf("unsupported server type: %s", serverType)
	}
}

// NewPHPVhost 创建 PHP 虚拟主机实例
func NewPHPVhost(serverType Type, configDir string) (types.PHPVhost, error) {
	switch serverType {
	case TypeNginx:
		return nginx.NewPHPVhost(configDir)
	case TypeApache:
		return apache.NewPHPVhost(configDir)
	default:
		return nil, fmt.Errorf("unsupported server type: %s", serverType)
	}
}

// NewProxyVhost 创建反向代理虚拟主机实例
func NewProxyVhost(serverType Type, configDir string) (types.ProxyVhost, error) {
	switch serverType {
	case TypeNginx:
		return nginx.NewProxyVhost(configDir)
	case TypeApache:
		return apache.NewProxyVhost(configDir)
	default:
		return nil, fmt.Errorf("unsupported server type: %s", serverType)
	}
}
