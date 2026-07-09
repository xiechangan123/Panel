package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/service"
)

// ToolboxSystemRoutes 工具箱-系统 路由
func ToolboxSystemRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.ToolboxSystemService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/toolbox_system/dns", Handler: svc.GetDNS},
		{Method: http.MethodPost, Path: "/api/toolbox_system/dns", Handler: svc.UpdateDNS},
		{Method: http.MethodGet, Path: "/api/toolbox_system/swap", Handler: svc.GetSWAP},
		{Method: http.MethodPost, Path: "/api/toolbox_system/swap", Handler: svc.UpdateSWAP},
		{Method: http.MethodGet, Path: "/api/toolbox_system/timezone", Handler: svc.GetTimezone},
		{Method: http.MethodPost, Path: "/api/toolbox_system/timezone", Handler: svc.UpdateTimezone},
		{Method: http.MethodPost, Path: "/api/toolbox_system/time", Handler: svc.UpdateTime},
		{Method: http.MethodPost, Path: "/api/toolbox_system/sync_time", Handler: svc.SyncTime},
		{Method: http.MethodGet, Path: "/api/toolbox_system/ntp_servers", Handler: svc.GetNTPServers},
		{Method: http.MethodPost, Path: "/api/toolbox_system/ntp_servers", Handler: svc.UpdateNTPServers},
		{Method: http.MethodGet, Path: "/api/toolbox_system/hostname", Handler: svc.GetHostname},
		{Method: http.MethodPost, Path: "/api/toolbox_system/hostname", Handler: svc.UpdateHostname},
		{Method: http.MethodGet, Path: "/api/toolbox_system/hosts", Handler: svc.GetHosts},
		{Method: http.MethodPost, Path: "/api/toolbox_system/hosts", Handler: svc.UpdateHosts},
	}, nil
}
