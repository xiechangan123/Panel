package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/service"
)

// ToolboxLogRoutes 工具箱-日志清理 路由
func ToolboxLogRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.ToolboxLogService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/toolbox_log/scan", Handler: svc.Scan},
		{Method: http.MethodPost, Path: "/api/toolbox_log/clean", Handler: svc.Clean},
	}, nil
}
