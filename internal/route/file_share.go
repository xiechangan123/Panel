package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
)

// FileShareRoutes 文件分享路由
func FileShareRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.FileShareService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/file_share", Handler: svc.List,
			Summary: "文件分享列表", Tags: []string{"文件"},
			Response: service.Envelope[[]*biz.FileShare]{}},
		{Method: http.MethodPost, Path: "/api/file_share", Handler: svc.Create,
			Summary: "创建文件分享", Tags: []string{"文件"},
			Request: request.FileShareCreate{}, Response: service.Envelope[biz.FileShare]{}},
		{Method: http.MethodDelete, Path: "/api/file_share/{id}", Handler: svc.Delete,
			Summary: "取消文件分享", Tags: []string{"文件"},
			Request: request.ID{}},
		// 顶层免登录下载
		{Method: http.MethodGet, Path: "/download/{token}", Handler: svc.Download},
	}, nil
}
