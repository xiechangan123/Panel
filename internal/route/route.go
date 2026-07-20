package route

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/libtnb/validator/contrib/openapi"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/middleware"
	"github.com/acepanel/panel/v3/internal/registry"
	"github.com/acepanel/panel/v3/pkg/config"
)

const RoutePrefix = "routes:"

var Package = do.Package(
	do.LazyNamed(RoutePrefix+"user", UserRoutes), do.LazyNamed(RoutePrefix+"user_passkey", UserPasskeyRoutes),
	do.LazyNamed(RoutePrefix+"user_token", UserTokenRoutes), do.LazyNamed(RoutePrefix+"safe", SafeRoutes),
	do.LazyNamed(RoutePrefix+"task", TaskRoutes), do.LazyNamed(RoutePrefix+"home", HomeRoutes),
	do.LazyNamed(RoutePrefix+"website", WebsiteRoutes), do.LazyNamed(RoutePrefix+"website_stat", WebsiteStatRoutes),
	do.LazyNamed(RoutePrefix+"project", ProjectRoutes), do.LazyNamed(RoutePrefix+"database", DatabaseRoutes),
	do.LazyNamed(RoutePrefix+"database_server", DatabaseServerRoutes), do.LazyNamed(RoutePrefix+"database_user", DatabaseUserRoutes),
	do.LazyNamed(RoutePrefix+"database_redis", DatabaseRedisRoutes), do.LazyNamed(RoutePrefix+"database_elasticsearch", DatabaseElasticsearchRoutes),
	do.LazyNamed(RoutePrefix+"cert", CertRoutes), do.LazyNamed(RoutePrefix+"backup", BackupRoutes),
	do.LazyNamed(RoutePrefix+"backup_storage", BackupStorageRoutes), do.LazyNamed(RoutePrefix+"app", AppRoutes),
	do.LazyNamed(RoutePrefix+"environment", EnvironmentRoutes), do.LazyNamed(RoutePrefix+"container", ContainerRoutes),
	do.LazyNamed(RoutePrefix+"file", FileRoutes), do.LazyNamed(RoutePrefix+"file_share", FileShareRoutes),
	do.LazyNamed(RoutePrefix+"cron", CronRoutes),
	do.LazyNamed(RoutePrefix+"process", ProcessRoutes), do.LazyNamed(RoutePrefix+"firewall", FirewallRoutes),
	do.LazyNamed(RoutePrefix+"ssh", SSHRoutes), do.LazyNamed(RoutePrefix+"systemctl", SystemctlRoutes),
	do.LazyNamed(RoutePrefix+"setting", SettingRoutes), do.LazyNamed(RoutePrefix+"log", LogRoutes),
	do.LazyNamed(RoutePrefix+"monitor", MonitorRoutes), do.LazyNamed(RoutePrefix+"webhook", WebHookRoutes),
	do.LazyNamed(RoutePrefix+"template", TemplateRoutes), do.LazyNamed(RoutePrefix+"toolbox_network", ToolboxNetworkRoutes),
	do.LazyNamed(RoutePrefix+"toolbox_system", ToolboxSystemRoutes), do.LazyNamed(RoutePrefix+"toolbox_benchmark", ToolboxBenchmarkRoutes),
	do.LazyNamed(RoutePrefix+"toolbox_ssh", ToolboxSSHRoutes), do.LazyNamed(RoutePrefix+"toolbox_disk", ToolboxDiskRoutes),
	do.LazyNamed(RoutePrefix+"toolbox_log", ToolboxLogRoutes), do.LazyNamed(RoutePrefix+"toolbox_migration", ToolboxMigrationRoutes),
	do.LazyNamed(RoutePrefix+"tamper", TamperRoutes),
	do.LazyNamed(RoutePrefix+"ws", WsRoutes),
)

// ThrottleRule 端点级限流规则。
type ThrottleRule struct {
	Tokens   uint64
	Interval time.Duration
}

// Endpoint 声明一个 HTTP 端点：如何服务，以及（经 Request/Response 样本）如何生成文档。
// 无 Request/Response 的端点（探针、WebSocket）不进 OpenAPI 文档。
type Endpoint struct {
	Method   string
	Path     string // 绝对路径，如 "/api/users"
	Handler  http.HandlerFunc
	Summary  string
	Tags     []string
	Request  any // request.* 样本
	Response any // service.Envelope[...] 样本
	Status   int
	Public   bool          // 登录白名单（MustLogin 放行）
	Throttle *ThrottleRule // 非 nil 时端点级限流
}

// Endpoints 是一个模块对 HTTP 路由的贡献。
type Endpoints []Endpoint

// HTTP 将全部 "routes:*" 贡献注册到 r。
func HTTP(i do.Injector, r chi.Router) error {
	conf := do.MustInvoke[*config.Config](i)

	groups, err := registry.Collect[Endpoints](i, RoutePrefix)
	if err != nil {
		return err
	}
	for _, endpoints := range groups {
		for _, e := range endpoints {
			if e.Throttle != nil {
				r.With(middleware.Throttle(conf.HTTP.IPHeader, e.Throttle.Tokens, e.Throttle.Interval)).Method(e.Method, e.Path, e.Handler)
			} else {
				r.Method(e.Method, e.Path, e.Handler)
			}
		}
	}

	return nil
}

// PublicPaths 收集去重后的登录白名单路径，供 MustLogin 中间件放行。
func PublicPaths(i do.Injector) ([]string, error) {
	groups, err := registry.Collect[Endpoints](i, RoutePrefix)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	paths := make([]string, 0)
	for _, endpoints := range groups {
		for _, e := range endpoints {
			if !e.Public {
				continue
			}
			if _, ok := seen[e.Path]; ok {
				continue
			}
			seen[e.Path] = struct{}{}
			paths = append(paths, e.Path)
		}
	}

	return paths, nil
}

// SpecJSON 从每个带文档样本的端点组装 OpenAPI 3 文档。
func SpecJSON(i do.Injector, title string) ([]byte, error) {
	g := openapi.New(title, buildVersion(),
		openapi.WithType(time.Time{}, &openapi.Schema{Type: "string", Format: "date-time"}),
	)

	groups, err := registry.Collect[Endpoints](i, RoutePrefix)
	if err != nil {
		return nil, err
	}
	for _, endpoints := range groups {
		for _, e := range endpoints {
			if e.Request == nil && e.Response == nil {
				continue
			}
			if err := g.Add(e.Method, e.Path, openapi.Op{
				Summary:  e.Summary,
				Tags:     e.Tags,
				Request:  e.Request,
				Response: e.Response,
				Status:   e.Status,
			}); err != nil {
				return nil, err
			}
		}
	}

	return g.JSON()
}

func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}
