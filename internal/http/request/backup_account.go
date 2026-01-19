package request

import "github.com/acepanel/panel/pkg/types"

type BackupAccountCreate struct {
	Type string                  `form:"type" json:"type" validate:"required|in:local,s3,sftp,webdav"`
	Name string                  `form:"name" json:"name" validate:"required"`
	Info types.BackupAccountInfo `form:"info" json:"info"`
}

type BackupAccountUpdate struct {
	ID   uint                    `form:"id" json:"id" validate:"required|exists:backup_accounts,id"`
	Type string                  `form:"type" json:"type" validate:"required|in:local,s3,sftp,webdav"`
	Name string                  `form:"name" json:"name" validate:"required"`
	Info types.BackupAccountInfo `form:"info" json:"info"`
}
