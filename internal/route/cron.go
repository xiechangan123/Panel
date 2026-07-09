package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
)

// CronRoutes 计划任务路由
func CronRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.CronService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/cron", Handler: svc.List,
			Summary: "计划任务列表", Tags: []string{"计划任务"},
			Request: request.Paginate{}, Response: service.Envelope[service.Page[*biz.Cron]]{}},
		{Method: http.MethodPost, Path: "/api/cron", Handler: svc.Create,
			Summary: "创建计划任务", Tags: []string{"计划任务"},
			Request: request.CronCreate{}},
		{Method: http.MethodPut, Path: "/api/cron/{id}", Handler: svc.Update,
			Summary: "更新计划任务", Tags: []string{"计划任务"},
			Request: request.CronUpdate{}},
		{Method: http.MethodGet, Path: "/api/cron/{id}", Handler: svc.Get,
			Summary: "获取计划任务", Tags: []string{"计划任务"},
			Request: request.ID{}, Response: service.Envelope[biz.Cron]{}},
		{Method: http.MethodDelete, Path: "/api/cron/{id}", Handler: svc.Delete,
			Summary: "删除计划任务", Tags: []string{"计划任务"},
			Request: request.ID{}},
		{Method: http.MethodPost, Path: "/api/cron/{id}/status", Handler: svc.Status,
			Summary: "设置计划任务状态", Tags: []string{"计划任务"},
			Request: request.CronStatus{}},
	}, nil
}
