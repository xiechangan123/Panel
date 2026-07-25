package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
)

// NotifyRoutes 通知渠道路由
func NotifyRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.NotifyService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/notify/channel", Handler: svc.List, Summary: "通知渠道列表", Tags: []string{"通知"}, Request: request.Paginate{}, Response: service.Envelope[service.Page[*biz.NotifyChannel]]{}},
		{Method: http.MethodGet, Path: "/api/notify/channel/all", Handler: svc.All, Summary: "全部通知渠道", Tags: []string{"通知"}, Response: service.Envelope[[]*biz.NotifyChannel]{}},
		{Method: http.MethodPost, Path: "/api/notify/channel", Handler: svc.Create, Summary: "创建通知渠道", Tags: []string{"通知"}, Request: request.NotifyChannelCreate{}, Response: service.Envelope[biz.NotifyChannel]{}},
		{Method: http.MethodGet, Path: "/api/notify/channel/{id}", Handler: svc.Get, Summary: "获取通知渠道", Tags: []string{"通知"}, Request: request.ID{}, Response: service.Envelope[biz.NotifyChannel]{}},
		{Method: http.MethodPut, Path: "/api/notify/channel/{id}", Handler: svc.Update, Summary: "更新通知渠道", Tags: []string{"通知"}, Request: request.NotifyChannelUpdate{}},
		{Method: http.MethodDelete, Path: "/api/notify/channel/{id}", Handler: svc.Delete, Summary: "删除通知渠道", Tags: []string{"通知"}, Request: request.ID{}},
		{Method: http.MethodPost, Path: "/api/notify/channel/{id}/test", Handler: svc.Test, Summary: "测试通知渠道", Tags: []string{"通知"}, Request: request.ID{}},
		{Method: http.MethodGet, Path: "/api/notify/setting", Handler: svc.GetSetting, Summary: "获取事件通知设置", Tags: []string{"通知"}, Response: service.Envelope[request.NotifySetting]{}},
		{Method: http.MethodPost, Path: "/api/notify/setting", Handler: svc.UpdateSetting, Summary: "更新事件通知设置", Tags: []string{"通知"}, Request: request.NotifySetting{}},
	}, nil
}
