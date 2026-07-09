package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
)

// DatabaseRoutes 数据库路由
func DatabaseRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.DatabaseService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/database", Handler: svc.List,
			Summary: "获取数据库列表", Tags: []string{"数据库"},
			Request: request.DatabaseList{}, Response: service.Envelope[service.Page[*biz.Database]]{}},
		{Method: http.MethodPost, Path: "/api/database", Handler: svc.Create,
			Summary: "创建数据库", Tags: []string{"数据库"},
			Request: request.DatabaseCreate{}},
		{Method: http.MethodDelete, Path: "/api/database", Handler: svc.Delete,
			Summary: "删除数据库", Tags: []string{"数据库"},
			Request: request.DatabaseDelete{}},
		{Method: http.MethodPost, Path: "/api/database/comment", Handler: svc.Comment,
			Summary: "设置数据库注释", Tags: []string{"数据库"},
			Request: request.DatabaseComment{}},
	}, nil
}
