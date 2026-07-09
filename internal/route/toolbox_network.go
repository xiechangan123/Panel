package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/service"
)

// ToolboxNetworkRoutes 工具箱-网络 路由
func ToolboxNetworkRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.ToolboxNetworkService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/toolbox_network/list", Handler: svc.List},
	}, nil
}
