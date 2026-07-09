package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
)

// TemplateRoutes 模板路由
func TemplateRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.TemplateService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/template", Handler: svc.List, Summary: "模板列表", Tags: []string{"模板"}},
		{Method: http.MethodGet, Path: "/api/template/{slug}", Handler: svc.Get, Summary: "获取模板", Tags: []string{"模板"}, Request: request.TemplateSlug{}},
		{Method: http.MethodPost, Path: "/api/template", Handler: svc.Create, Summary: "使用模板创建编排", Tags: []string{"模板"}, Request: request.TemplateCreate{}},
		{Method: http.MethodPost, Path: "/api/template/{slug}/callback", Handler: svc.Callback, Summary: "模板下载回调", Tags: []string{"模板"}, Request: request.TemplateSlug{}},
	}, nil
}
