package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/types"
)

// ProjectRoutes 项目相关路由
func ProjectRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.ProjectService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/project", Handler: svc.List,
			Summary: "项目列表", Tags: []string{"项目"},
			Request: request.Paginate{}, Response: service.Envelope[service.Page[*types.ProjectDetail]]{}},
		{Method: http.MethodPost, Path: "/api/project", Handler: svc.Create,
			Summary: "创建项目", Tags: []string{"项目"},
			Request: request.ProjectCreate{}, Response: service.Envelope[types.ProjectDetail]{}},
		{Method: http.MethodGet, Path: "/api/project/{id}", Handler: svc.Get,
			Summary: "获取项目详情", Tags: []string{"项目"},
			Request: request.ID{}, Response: service.Envelope[types.ProjectDetail]{}},
		{Method: http.MethodPut, Path: "/api/project/{id}", Handler: svc.Update,
			Summary: "更新项目", Tags: []string{"项目"}, Request: request.ProjectUpdate{}},
		{Method: http.MethodDelete, Path: "/api/project/{id}", Handler: svc.Delete,
			Summary: "删除项目", Tags: []string{"项目"}, Request: request.ID{}},
	}, nil
}
