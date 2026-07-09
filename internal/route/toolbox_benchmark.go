package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/service"
)

// ToolboxBenchmarkRoutes 工具箱-跑分 路由
func ToolboxBenchmarkRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.ToolboxBenchmarkService](i)

	return Endpoints{
		{Method: http.MethodPost, Path: "/api/toolbox_benchmark/test", Handler: svc.Test},
	}, nil
}
