package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
)

// SSHRoutes SSH 路由
func SSHRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.SSHService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/ssh", Handler: svc.List,
			Summary: "SSH 列表", Tags: []string{"SSH"},
			Request: request.Paginate{}, Response: service.Envelope[service.Page[*biz.SSH]]{}},
		{Method: http.MethodPost, Path: "/api/ssh", Handler: svc.Create,
			Summary: "创建 SSH", Tags: []string{"SSH"},
			Request: request.SSHCreate{}},
		{Method: http.MethodPut, Path: "/api/ssh/{id}", Handler: svc.Update,
			Summary: "更新 SSH", Tags: []string{"SSH"},
			Request: request.SSHUpdate{}},
		{Method: http.MethodGet, Path: "/api/ssh/{id}", Handler: svc.Get,
			Summary: "获取 SSH", Tags: []string{"SSH"},
			Request: request.ID{}, Response: service.Envelope[biz.SSH]{}},
		{Method: http.MethodDelete, Path: "/api/ssh/{id}", Handler: svc.Delete,
			Summary: "删除 SSH", Tags: []string{"SSH"},
			Request: request.ID{}},
	}, nil
}
