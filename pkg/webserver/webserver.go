package webserver

import (
	"fmt"
	"slices"

	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/webserver/apache"
	"github.com/acepanel/panel/v3/pkg/webserver/nginx"
	"github.com/acepanel/panel/v3/pkg/webserver/openlitespeed"
	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

// dialects 已注册的 Web 服务器方言
var dialects = map[Type]types.Dialect{
	TypeNginx:         nginx.Dialect{},
	TypeApache:        apache.Dialect{},
	TypeOpenLiteSpeed: openlitespeed.Dialect{},
}

// Dialect 在具体方言之上补充与服务器无关的通用逻辑
type Dialect struct {
	types.Dialect
	Type Type
}

// Get 按类型取方言
func Get(t Type) (Dialect, error) {
	d, ok := dialects[t]
	if !ok {
		return Dialect{}, fmt.Errorf("unsupported web server: %s", t)
	}

	return Dialect{Dialect: d, Type: t}, nil
}

// Types 已注册的 Web 服务器类型
func Types() []Type {
	keys := make([]Type, 0, len(dialects))
	for t := range dialects {
		keys = append(keys, t)
	}
	slices.Sort(keys)
	return keys
}

// NewVhost 按网站类型构造站点 vhost，typ 取值 proxy、php、static
func (d Dialect) NewVhost(typ, configDir string) (types.Vhost, error) {
	switch typ {
	case "proxy":
		return d.NewProxyVhost(configDir)
	case "php":
		return d.NewPHPVhost(configDir)
	case "static":
		return d.NewStaticVhost(configDir)
	default:
		return nil, fmt.Errorf("unsupported website type: %s", typ)
	}
}

// Reload 重载服务，失败时附带配置测试输出
func (d Dialect) Reload() error {
	if err := d.BeforeReload(); err != nil {
		return err
	}

	return d.reload()
}

// ReloadIfRunning 仅在服务运行时重载，未运行时配置会在下次启动时生效
func (d Dialect) ReloadIfRunning() error {
	if err := d.BeforeReload(); err != nil {
		return err
	}
	if running, _ := systemctl.Status(d.Service()); !running {
		return nil
	}

	return d.reload()
}

func (d Dialect) reload() error {
	if err := systemctl.Reload(d.Service()); err != nil {
		out, _ := shell.Execf(d.ConfigTest())
		return fmt.Errorf("failed to reload %s: %w; config test: %s", d.Service(), err, out)
	}

	return nil
}
