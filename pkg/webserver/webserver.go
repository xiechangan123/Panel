package webserver

import (
	"fmt"

	"github.com/acepanel/panel/pkg/webserver/apache"
	"github.com/acepanel/panel/pkg/webserver/nginx"
	"github.com/acepanel/panel/pkg/webserver/types"
)

// NewVhost 创建虚拟主机管理实例
func NewVhost(serverType Type, configDir string) (types.Vhost, error) {
	switch serverType {
	case TypeNginx:
		return nginx.NewVhost(configDir)
	case TypeApache:
		return apache.NewVhost(configDir)
	default:
		return nil, fmt.Errorf("unsupported server type: %s", serverType)
	}
}
