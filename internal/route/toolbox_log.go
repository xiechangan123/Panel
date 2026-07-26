package route

import (
	"net/http"

	"github.com/acepanel/panel/v3/internal/service"
)

// ToolboxLogRoutes 工具箱-日志清理 路由
func ToolboxLogRoutes(toolboxLogService *service.ToolboxLogService) Endpoints {
	svc := toolboxLogService

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/toolbox_log/scan", Handler: svc.Scan},
		{Method: http.MethodPost, Path: "/api/toolbox_log/clean", Handler: svc.Clean},
	}
}
