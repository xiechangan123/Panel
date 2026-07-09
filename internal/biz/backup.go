package biz

import (
	"context"

	"github.com/acepanel/panel/v3/pkg/types"
)

type BackupType string

const (
	BackupTypePath       BackupType = "path"
	BackupTypeWebsite    BackupType = "website"
	BackupTypeMySQL      BackupType = "mysql"
	BackupTypePostgres   BackupType = "postgresql"
	BackupTypeClickHouse BackupType = "clickhouse"
	BackupTypeRedis      BackupType = "redis"
	BackupTypeValkey     BackupType = "valkey"
	BackupTypePanel      BackupType = "panel"
)

type BackupRepo interface {
	List(typ BackupType) ([]*types.BackupFile, error)
	Create(ctx context.Context, typ BackupType, target string, account uint) error
	CreatePanel() error
	Delete(ctx context.Context, typ BackupType, name string) error
	Restore(ctx context.Context, typ BackupType, backup, target string) error
	ClearExpired(path, prefix string, save uint) error
	ClearStorageExpired(account uint, typ BackupType, prefix string, save uint) error
	CutoffLog(path, target string) (string, error)
	CutoffUpload(account uint, typ BackupType, name string, files []string) error
	GetDefaultPath(typ BackupType) string
	FixPanel() error
	UpdatePanel(version, url, checksum string, progress func(string)) error
}

type BackupUsecase struct {
	repo BackupRepo
}

func NewBackupUsecase(repo BackupRepo) *BackupUsecase {
	return &BackupUsecase{repo: repo}
}

func (uc *BackupUsecase) List(typ BackupType) ([]*types.BackupFile, error) {
	return uc.repo.List(typ)
}

func (uc *BackupUsecase) Create(ctx context.Context, typ BackupType, target string, account uint) error {
	return uc.repo.Create(ctx, typ, target, account)
}

func (uc *BackupUsecase) CreatePanel() error {
	return uc.repo.CreatePanel()
}

func (uc *BackupUsecase) Delete(ctx context.Context, typ BackupType, name string) error {
	return uc.repo.Delete(ctx, typ, name)
}

func (uc *BackupUsecase) Restore(ctx context.Context, typ BackupType, backup, target string) error {
	return uc.repo.Restore(ctx, typ, backup, target)
}

func (uc *BackupUsecase) ClearExpired(path, prefix string, save uint) error {
	return uc.repo.ClearExpired(path, prefix, save)
}

func (uc *BackupUsecase) ClearStorageExpired(account uint, typ BackupType, prefix string, save uint) error {
	return uc.repo.ClearStorageExpired(account, typ, prefix, save)
}

func (uc *BackupUsecase) CutoffLog(path, target string) (string, error) {
	return uc.repo.CutoffLog(path, target)
}

func (uc *BackupUsecase) CutoffUpload(account uint, typ BackupType, name string, files []string) error {
	return uc.repo.CutoffUpload(account, typ, name, files)
}

func (uc *BackupUsecase) GetDefaultPath(typ BackupType) string {
	return uc.repo.GetDefaultPath(typ)
}

func (uc *BackupUsecase) FixPanel() error {
	return uc.repo.FixPanel()
}

func (uc *BackupUsecase) UpdatePanel(version, url, checksum string, progress func(string)) error {
	return uc.repo.UpdatePanel(version, url, checksum, progress)
}
