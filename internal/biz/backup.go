package biz

import (
	"context"
	"time"
)

type BackupType string

const (
	BackupTypePath     BackupType = "path"
	BackupTypeWebsite  BackupType = "website"
	BackupTypeMySQL    BackupType = "mysql"
	BackupTypePostgres BackupType = "postgres"
	BackupTypeRedis    BackupType = "redis"
	BackupTypePanel    BackupType = "panel"
)

type Backup struct {
	ID        uint       `gorm:"primaryKey" json:"id"`                 // 备份 ID
	AccountID uint       `gorm:"not null;default:0" json:"account_id"` // 关联的备份账号 ID
	Type      BackupType `gorm:"not null;default:''" json:"type"`      // 备份类型
	Name      string     `gorm:"not null;default:''" json:"name"`      // 备份文件名
	Size      int64      `gorm:"not null;default:0" json:"size"`       // 备份文件大小
	Status    bool       `gorm:"not null;default:false" json:"status"` // 备份状态
	Log       string     `gorm:"not null;default:''" json:"log"`       // 备份日志
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	Account *BackupAccount `gorm:"foreignKey:AccountID" json:"account"`
}

type BackupRepo interface {
	List(page, limit uint, typ BackupType) ([]*Backup, int64, error)
	Create(ctx context.Context, typ BackupType, target string, account uint) error
	CreatePanel() error
	Delete(ctx context.Context, id uint) error
	Restore(ctx context.Context, id uint, target string) error
	ClearExpired(path, prefix string, save int) error
	ClearAccountExpired(account uint, typ BackupType, prefix string, save int) error
	CutoffLog(path, target string) error
	GetDefaultPath(typ BackupType) string
	FixPanel() error
	UpdatePanel(version, url, checksum string) error
}
