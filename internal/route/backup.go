package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/types"
)

// BackupRoutes 备份路由
func BackupRoutes(i do.Injector) (Endpoints, error) {
	backup := do.MustInvoke[*service.BackupService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/backup/{type}", Handler: backup.List, Summary: "备份列表", Tags: []string{"备份"},
			Request: request.BackupList{}, Response: service.Envelope[service.Page[*types.BackupFile]]{}},
		{Method: http.MethodPost, Path: "/api/backup/{type}", Handler: backup.Create, Summary: "创建备份", Tags: []string{"备份"},
			Request: request.BackupCreate{}},
		{Method: http.MethodPost, Path: "/api/backup/{type}/upload", Handler: backup.Upload, Summary: "上传备份", Tags: []string{"备份"}},
		{Method: http.MethodDelete, Path: "/api/backup/{type}/delete", Handler: backup.Delete, Summary: "删除备份", Tags: []string{"备份"},
			Request: request.BackupFile{}},
		{Method: http.MethodPost, Path: "/api/backup/{type}/restore", Handler: backup.Restore, Summary: "恢复备份", Tags: []string{"备份"},
			Request: request.BackupRestore{}},
	}, nil
}
