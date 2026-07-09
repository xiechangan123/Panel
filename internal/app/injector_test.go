package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/require"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/bootstrap"
	"github.com/acepanel/panel/v3/internal/command"
	"github.com/acepanel/panel/v3/internal/injector"
	"github.com/acepanel/panel/v3/internal/job"
	"github.com/acepanel/panel/v3/internal/registry"
	"github.com/acepanel/panel/v3/internal/route"
	"github.com/acepanel/panel/v3/pkg/config"
)

// TestContainer 构建完整对象图，在测试期而非启动期暴露装配错误
func TestContainer(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "panel/storage/logs"), 0o755))

	// 直接构造内存配置并注入
	conf := &config.Config{
		App: config.AppConfig{
			Debug:    true,
			Key:      "12345678901234567890123456789012",
			Locale:   "zh_CN",
			Timezone: "UTC",
			Root:     tmp,
		},
		HTTP:    config.HTTPConfig{Port: 8888, TLS: "off"},
		Session: config.SessionConfig{Lifetime: 120},
	}
	require.NoError(t, bootstrap.InitGlobal(conf))

	inj := injector.New()
	defer func() { _ = inj.Shutdown() }()

	// 覆盖配置提供者，跳过 config.Load 与全局副作用
	do.OverrideValue(inj, conf)

	// 构造两个入口即触发全图：路由 + SpecJSON + cron 注册（5 段表达式）+ nil 证书重载器
	_, err := do.Invoke[*app.Ace](inj)
	require.NoError(t, err)
	_, err = do.Invoke[*app.Cli](inj)
	require.NoError(t, err)

	// 每个带冒号的命名贡献必须命中已知前缀（拼错如 route:user 会被静默丢弃）
	require.NoError(t, registry.Verify(inj, route.RoutePrefix, command.Prefix, job.Prefix))

	routes, err := registry.Collect[route.Endpoints](inj, route.RoutePrefix)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	cmds, err := command.Commands(inj)
	require.NoError(t, err)
	require.NotEmpty(t, cmds)

	jobs, err := registry.Collect[job.Job](inj, job.Prefix)
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	spec, err := route.SpecJSON(inj, "AcePanel")
	require.NoError(t, err)
	require.NotEmpty(t, spec)
}
