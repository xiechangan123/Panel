package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
)

// WebHookRoutes WebHook 管理与回调路由
func WebHookRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.WebHookService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/webhook", Handler: svc.List, Summary: "WebHook 列表", Tags: []string{"WebHook"}, Request: request.Paginate{}, Response: service.Envelope[service.Page[*biz.WebHook]]{}},
		{Method: http.MethodPost, Path: "/api/webhook", Handler: svc.Create, Summary: "创建 WebHook", Tags: []string{"WebHook"}, Request: request.WebHookCreate{}, Response: service.Envelope[biz.WebHook]{}},
		{Method: http.MethodPut, Path: "/api/webhook/{id}", Handler: svc.Update, Summary: "更新 WebHook", Tags: []string{"WebHook"}, Request: request.WebHookUpdate{}},
		{Method: http.MethodGet, Path: "/api/webhook/{id}", Handler: svc.Get, Summary: "获取 WebHook", Tags: []string{"WebHook"}, Request: request.ID{}, Response: service.Envelope[biz.WebHook]{}},
		{Method: http.MethodDelete, Path: "/api/webhook/{id}", Handler: svc.Delete, Summary: "删除 WebHook", Tags: []string{"WebHook"}, Request: request.ID{}},
		// 顶层回调
		{Method: http.MethodGet, Path: "/webhook/{key}", Handler: svc.Call},
		{Method: http.MethodPost, Path: "/webhook/{key}", Handler: svc.Call},
	}, nil
}
