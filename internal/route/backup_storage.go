package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
)

// BackupStorageRoutes 备份存储路由
func BackupStorageRoutes(i do.Injector) (Endpoints, error) {
	backupStorage := do.MustInvoke[*service.BackupStorageService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/backup_storage", Handler: backupStorage.List, Summary: "备份存储列表", Tags: []string{"备份存储"},
			Request: request.Paginate{}, Response: service.Envelope[service.Page[*biz.BackupStorage]]{}},
		{Method: http.MethodPost, Path: "/api/backup_storage", Handler: backupStorage.Create, Summary: "创建备份存储", Tags: []string{"备份存储"},
			Request: request.BackupStorageCreate{}, Response: service.Envelope[biz.BackupStorage]{}},
		{Method: http.MethodPut, Path: "/api/backup_storage/{id}", Handler: backupStorage.Update, Summary: "更新备份存储", Tags: []string{"备份存储"},
			Request: request.BackupStorageUpdate{}},
		{Method: http.MethodGet, Path: "/api/backup_storage/{id}", Handler: backupStorage.Get, Summary: "获取备份存储", Tags: []string{"备份存储"},
			Request: request.ID{}, Response: service.Envelope[biz.BackupStorage]{}},
		{Method: http.MethodDelete, Path: "/api/backup_storage/{id}", Handler: backupStorage.Delete, Summary: "删除备份存储", Tags: []string{"备份存储"},
			Request: request.ID{}},
	}, nil
}
