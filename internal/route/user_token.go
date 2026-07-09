package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
)

// UserTokenRoutes 用户令牌路由
func UserTokenRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.UserTokenService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/user_tokens", Handler: svc.List, Summary: "获取用户令牌列表", Tags: []string{"用户令牌"}, Request: request.UserTokenList{}, Response: service.Envelope[service.Page[*biz.UserToken]]{}},
		{Method: http.MethodPost, Path: "/api/user_tokens", Handler: svc.Create, Summary: "创建用户令牌", Tags: []string{"用户令牌"}, Request: request.UserTokenCreate{}, Response: service.Envelope[biz.UserToken]{}},
		{Method: http.MethodPut, Path: "/api/user_tokens/{id}", Handler: svc.Update, Summary: "更新用户令牌", Tags: []string{"用户令牌"}, Request: request.UserTokenUpdate{}, Response: service.Envelope[biz.UserToken]{}},
		{Method: http.MethodDelete, Path: "/api/user_tokens/{id}", Handler: svc.Delete, Summary: "删除用户令牌", Tags: []string{"用户令牌"}, Request: request.ID{}},
	}, nil
}
