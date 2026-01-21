package request

import "github.com/acepanel/panel/pkg/types"

type BackupStorageCreate struct {
	Type string                  `form:"type" json:"type" validate:"required|in:s3,sftp,webdav"`
	Name string                  `form:"name" json:"name" validate:"required"`
	Info types.BackupStorageInfo `form:"info" json:"info"`
}

type BackupStorageUpdate struct {
	ID   uint                    `form:"id" json:"id" validate:"required|exists:backup_storages,id"`
	Type string                  `form:"type" json:"type" validate:"required|in:s3,sftp,webdav"`
	Name string                  `form:"name" json:"name" validate:"required"`
	Info types.BackupStorageInfo `form:"info" json:"info"`
}
