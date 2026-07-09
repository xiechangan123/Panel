package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/require"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/command"
	"github.com/acepanel/panel/v3/internal/injector"
	"github.com/acepanel/panel/v3/internal/job"
	"github.com/acepanel/panel/v3/internal/registry"
	"github.com/acepanel/panel/v3/internal/route"
)

// TestContainer 构建完整对象图，在测试期而非启动期暴露装配错误
func TestContainer(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "panel/storage/logs"), 0o755))

	// 写入临时配置并经 PANEL_CONFIG 指向，避免读取 /opt/ace
	cfg := filepath.Join(tmp, "config.yml")
	content := "app:\n" +
		"  debug: true\n" +
		"  key: 12345678901234567890123456789012\n" +
		"  locale: zh_CN\n" +
		"  timezone: UTC\n" +
		"  root: " + tmp + "\n" +
		"http:\n" +
		"  port: 8888\n" +
		"  tls: \"off\"\n" +
		"database: {}\n" +
		"session:\n" +
		"  lifetime: 120\n"
	require.NoError(t, os.WriteFile(cfg, []byte(content), 0o600))
	t.Setenv("PANEL_CONFIG", cfg)

	inj := injector.New()
	defer func() { _ = inj.Shutdown() }()

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

	jobs, err := registry.Collect[job.JobFn](inj, job.Prefix)
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	spec, err := route.SpecJSON(inj, "AcePanel")
	require.NoError(t, err)
	require.NotEmpty(t, spec)
}
