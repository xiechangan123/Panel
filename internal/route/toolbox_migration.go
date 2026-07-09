package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/service"
)

// ToolboxMigrationRoutes 工具箱-迁移 路由
func ToolboxMigrationRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.ToolboxMigrationService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/toolbox_migration/status", Handler: svc.GetStatus},
		{Method: http.MethodPost, Path: "/api/toolbox_migration/precheck", Handler: svc.PreCheck},
		{Method: http.MethodGet, Path: "/api/toolbox_migration/items", Handler: svc.GetItems},
		{Method: http.MethodPost, Path: "/api/toolbox_migration/start", Handler: svc.Start},
		{Method: http.MethodPost, Path: "/api/toolbox_migration/reset", Handler: svc.Reset},
		{Method: http.MethodGet, Path: "/api/toolbox_migration/results", Handler: svc.GetResults},
		{Method: http.MethodGet, Path: "/api/toolbox_migration/log", Handler: svc.DownloadLog},
		{Method: http.MethodPost, Path: "/api/toolbox_migration/exec", Handler: svc.Exec},
	}, nil
}
