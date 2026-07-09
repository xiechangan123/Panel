package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/service"
)

// ToolboxDiskRoutes 工具箱-磁盘 路由
func ToolboxDiskRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.ToolboxDiskService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/toolbox_disk/list", Handler: svc.List},
		{Method: http.MethodPost, Path: "/api/toolbox_disk/partitions", Handler: svc.GetPartitions},
		{Method: http.MethodPost, Path: "/api/toolbox_disk/mount", Handler: svc.Mount},
		{Method: http.MethodPost, Path: "/api/toolbox_disk/umount", Handler: svc.Umount},
		{Method: http.MethodPost, Path: "/api/toolbox_disk/format", Handler: svc.Format},
		{Method: http.MethodPost, Path: "/api/toolbox_disk/init", Handler: svc.Init},
		{Method: http.MethodGet, Path: "/api/toolbox_disk/fstab", Handler: svc.GetFstab},
		{Method: http.MethodDelete, Path: "/api/toolbox_disk/fstab", Handler: svc.DeleteFstab},
		{Method: http.MethodGet, Path: "/api/toolbox_disk/lvm", Handler: svc.GetLVMInfo},
		{Method: http.MethodPost, Path: "/api/toolbox_disk/lvm/pv", Handler: svc.CreatePV},
		{Method: http.MethodDelete, Path: "/api/toolbox_disk/lvm/pv", Handler: svc.RemovePV},
		{Method: http.MethodPost, Path: "/api/toolbox_disk/lvm/vg", Handler: svc.CreateVG},
		{Method: http.MethodDelete, Path: "/api/toolbox_disk/lvm/vg", Handler: svc.RemoveVG},
		{Method: http.MethodPost, Path: "/api/toolbox_disk/lvm/lv", Handler: svc.CreateLV},
		{Method: http.MethodDelete, Path: "/api/toolbox_disk/lvm/lv", Handler: svc.RemoveLV},
		{Method: http.MethodPost, Path: "/api/toolbox_disk/lvm/lv/extend", Handler: svc.ExtendLV},
		{Method: http.MethodGet, Path: "/api/toolbox_disk/smart/disks", Handler: svc.GetSmartDisks},
		{Method: http.MethodGet, Path: "/api/toolbox_disk/smart/info", Handler: svc.GetSmartInfo},
		{Method: http.MethodGet, Path: "/api/toolbox_disk/raid/info", Handler: svc.GetRaidInfo},
	}, nil
}
