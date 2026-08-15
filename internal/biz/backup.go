package biz

import (
	"context"
	"log/slog"
	"time"

	"github.com/leonelquinteros/gotext"

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
	GetStorage(id uint) (*BackupStorage, error)
	Create(ctx context.Context, typ BackupType, target string, account uint) error
	CreatePanel() error
	Delete(typ BackupType, name string) error
	Restore(typ BackupType, backup, target string) error
	ClearExpired(path, prefix string, save uint) error
	ClearStorageExpired(account uint, dir, prefix string, save uint) error
	CutoffLog(path, target string) (string, error)
	CutoffUpload(account uint, typ BackupType, name string, files []string) error
	GetDefaultPath(typ BackupType) string
	FixPanel() error
	UpdatePanel(version, url, checksum string, progress func(string)) error
}

type BackupUsecase struct {
	repo   BackupRepo
	log    *slog.Logger
	notify *NotifyUsecase
	t      *gotext.Locale
}

func NewBackupUsecase(notifyUsecase *NotifyUsecase, t *gotext.Locale, log *slog.Logger, backupRepo BackupRepo) *BackupUsecase {
	return &BackupUsecase{
		repo:   backupRepo,
		log:    log,
		notify: notifyUsecase,
		t:      t,
	}
}

func (uc *BackupUsecase) List(typ BackupType) ([]*types.BackupFile, error) {
	return uc.repo.List(typ)
}

func (uc *BackupUsecase) Create(ctx context.Context, typ BackupType, target string, account uint) error {
	err := uc.repo.Create(ctx, typ, target, account)
	if err == nil {
		return nil
	}

	// 定时备份由 CLI 执行，命令返回即退出，异步通知来不及发出，必须同步发送
	if sendErr := uc.notify.SendEventSync(ctx, NotifyEventBackup, uc.t.Get("[AcePanel] Backup Failed"), NotifyBody(uc.t.Get("backup task failed"), [][2]string{
		{uc.t.Get("Type"), string(typ)},
		{uc.t.Get("Target"), target},
		{uc.t.Get("Error"), err.Error()},
		{uc.t.Get("Time"), time.Now().Format(time.DateTime)},
	})); sendErr != nil {
		uc.log.Warn("failed to send backup failure notification", slog.Any("err", sendErr))
	}

	return err
}

func (uc *BackupUsecase) CreatePanel() error {
	return uc.repo.CreatePanel()
}

func (uc *BackupUsecase) Delete(ctx context.Context, typ BackupType, name string) error {
	if err := uc.repo.Delete(typ, name); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("backup deleted", slog.String("type", OperationTypeBackup), slog.Uint64("operator_id", operatorID(ctx)), slog.String("backup_type", string(typ)), slog.String("name", name))

	return nil
}

func (uc *BackupUsecase) Restore(ctx context.Context, typ BackupType, backup, target string) error {
	if err := uc.repo.Restore(typ, backup, target); err != nil {
		return err
	}

	// 记录日志
	uc.log.Info("backup restored",
		slog.String("type", OperationTypeBackup),
		slog.Uint64("operator_id", operatorID(ctx)),
		slog.String("backup_type", string(typ)),
		slog.String("target", target),
	)

	return nil
}

func (uc *BackupUsecase) ClearExpired(path, prefix string, save uint) error {
	return uc.repo.ClearExpired(path, prefix, save)
}

func (uc *BackupUsecase) ClearStorageExpired(account uint, dir, prefix string, save uint) error {
	return uc.repo.ClearStorageExpired(account, dir, prefix, save)
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
