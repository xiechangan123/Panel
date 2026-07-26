package route

import (
	"net/http"

	"github.com/acepanel/panel/v3/internal/service"
)

// ToolboxNetworkRoutes 工具箱-网络 路由
func ToolboxNetworkRoutes(toolboxNetworkService *service.ToolboxNetworkService) Endpoints {
	svc := toolboxNetworkService

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/toolbox_network/list", Handler: svc.List},
	}
}
