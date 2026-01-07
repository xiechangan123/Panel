package webserver

import (
	"fmt"

	"github.com/acepanel/panel/pkg/webserver/apache"
	"github.com/acepanel/panel/pkg/webserver/nginx"
	"github.com/acepanel/panel/pkg/webserver/types"
)

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
