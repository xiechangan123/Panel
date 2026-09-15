package webserver

import (
	"context"
	"fmt"
	"slices"

	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/webserver/apache"
	"github.com/acepanel/panel/v3/pkg/webserver/caddy"
	"github.com/acepanel/panel/v3/pkg/webserver/nginx"
	"github.com/acepanel/panel/v3/pkg/webserver/openlitespeed"
	"github.com/acepanel/panel/v3/pkg/webserver/types"
)

// dialects 已注册的 Web 服务器方言
var dialects = map[Type]types.Dialect{
	TypeNginx:         nginx.Dialect{},
	TypeApache:        apache.Dialect{},
	TypeOpenLiteSpeed: openlitespeed.Dialect{},
	TypeCaddy:         caddy.Dialect{},
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
func (d Dialect) Reload(ctx context.Context) error {
	if err := d.BeforeReload(); err != nil {
		return err
	}

	return d.reload(ctx)
}

// ReloadIfRunning 仅在服务运行时重载，未运行时配置会在下次启动时生效
func (d Dialect) ReloadIfRunning(ctx context.Context) error {
	if err := d.BeforeReload(); err != nil {
		return err
	}
	// 查不到状态和确认未运行是两回事，前者不能当成「不用 reload」放过
	running, err := systemctl.Status(ctx, d.Service())
	if err != nil {
		return err
	}
	if !running {
		return nil
	}

	return d.reload(ctx)
}

// Test 执行配置检查并返回输出，各方言的检查命令都已合并 stderr
func (d Dialect) Test(ctx context.Context) (string, error) {
	return shell.Exec(ctx, d.ConfigTest())
}

func (d Dialect) reload(ctx context.Context) error {
	if err := systemctl.Reload(ctx, d.Service()); err != nil {
		out, _ := d.Test(ctx)
		return fmt.Errorf("failed to reload %s: %w; config test: %s", d.Service(), err, out)
	}

	return nil
}
