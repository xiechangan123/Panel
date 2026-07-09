package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
)

// TaskRoutes 后台任务路由
func TaskRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.TaskService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/task/status", Handler: svc.Status, Summary: "获取任务运行状态", Tags: []string{"任务"}},
		{Method: http.MethodGet, Path: "/api/task", Handler: svc.List, Summary: "获取任务列表", Tags: []string{"任务"}, Request: request.Paginate{}, Response: service.Envelope[service.Page[*biz.Task]]{}},
		{Method: http.MethodGet, Path: "/api/task/{id}", Handler: svc.Get, Summary: "获取任务详情", Tags: []string{"任务"}, Request: request.ID{}, Response: service.Envelope[biz.Task]{}},
		{Method: http.MethodDelete, Path: "/api/task/{id}", Handler: svc.Delete, Summary: "删除任务", Tags: []string{"任务"}, Request: request.ID{}},
	}, nil
}
