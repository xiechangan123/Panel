package biz

import (
	"context"
	"time"

	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/types"
)

type BackupAccountType string

const (
	BackupAccountTypeLocal  BackupAccountType = "local"
	BackupAccountTypeS3     BackupAccountType = "s3"
	BackupAccountTypeSFTP   BackupAccountType = "sftp"
	BackupAccountTypeWebDav BackupAccountType = "webdav"
)

type BackupAccount struct {
	ID        uint                    `gorm:"primaryKey" json:"id"`
	Type      BackupAccountType       `gorm:"not null;default:''" json:"type"`
	Name      string                  `gorm:"not null;default:''" json:"name"`
	Info      types.BackupAccountInfo `gorm:"not null;default:'{}';serializer:json" json:"info"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

type BackupAccountRepo interface {
	List(page, limit uint) ([]*BackupAccount, int64, error)
	Get(id uint) (*BackupAccount, error)
	Create(ctx context.Context, req *request.BackupAccountCreate) (*BackupAccount, error)
	Update(ctx context.Context, req *request.BackupAccountUpdate) error
	Delete(ctx context.Context, id uint) error
}
