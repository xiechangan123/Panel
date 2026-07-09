package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
)

// DatabaseUserRoutes 数据库用户路由
func DatabaseUserRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.DatabaseUserService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/database_user", Handler: svc.List,
			Summary: "获取用户列表", Tags: []string{"数据库用户"},
			Request: request.DatabaseList{}, Response: service.Envelope[service.Page[*biz.DatabaseUser]]{}},
		{Method: http.MethodPost, Path: "/api/database_user", Handler: svc.Create,
			Summary: "创建用户", Tags: []string{"数据库用户"},
			Request: request.DatabaseUserCreate{}},
		{Method: http.MethodGet, Path: "/api/database_user/{id}", Handler: svc.Get,
			Summary: "获取用户", Tags: []string{"数据库用户"},
			Request: request.ID{}, Response: service.Envelope[biz.DatabaseUser]{}},
		{Method: http.MethodPut, Path: "/api/database_user/{id}", Handler: svc.Update,
			Summary: "更新用户", Tags: []string{"数据库用户"},
			Request: request.DatabaseUserUpdate{}},
		{Method: http.MethodPut, Path: "/api/database_user/{id}/remark", Handler: svc.UpdateRemark,
			Summary: "更新用户备注", Tags: []string{"数据库用户"},
			Request: request.DatabaseUserUpdateRemark{}},
		{Method: http.MethodDelete, Path: "/api/database_user/{id}", Handler: svc.Delete,
			Summary: "删除用户", Tags: []string{"数据库用户"},
			Request: request.ID{}},
	}, nil
}
