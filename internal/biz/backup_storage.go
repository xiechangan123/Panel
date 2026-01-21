package biz

import (
	"context"
	"time"

	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/types"
)

type BackupStorageType string

const (
	BackupStorageTypeLocal  BackupStorageType = "local"
	BackupStorageTypeS3     BackupStorageType = "s3"
	BackupStorageTypeSFTP   BackupStorageType = "sftp"
	BackupStorageTypeWebDAV BackupStorageType = "webdav"
)

type BackupStorage struct {
	ID        uint                    `gorm:"primaryKey" json:"id"`
	Type      BackupStorageType       `gorm:"not null;default:''" json:"type"`
	Name      string                  `gorm:"not null;default:''" json:"name"`
	Info      types.BackupStorageInfo `gorm:"not null;default:'{}';serializer:json" json:"info"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

type BackupAccountRepo interface {
	List(page, limit uint) ([]*BackupStorage, int64, error)
	Get(id uint) (*BackupStorage, error)
	Create(ctx context.Context, req *request.BackupStorageCreate) (*BackupStorage, error)
	Update(ctx context.Context, req *request.BackupStorageUpdate) error
	Delete(ctx context.Context, id uint) error
}
