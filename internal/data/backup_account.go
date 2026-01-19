package data

import (
	"context"
	"log/slog"

	"github.com/leonelquinteros/gotext"
	"gorm.io/gorm"

	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/internal/http/request"
)

type backupAccountRepo struct {
	t   *gotext.Locale
	db  *gorm.DB
	log *slog.Logger
}

func NewBackupAccountRepo(t *gotext.Locale, db *gorm.DB, log *slog.Logger) biz.BackupAccountRepo {
	return &backupAccountRepo{
		t:   t,
		db:  db,
		log: log,
	}
}

func (r backupAccountRepo) List(page, limit uint) ([]*biz.BackupAccount, int64, error) {
	accounts := make([]*biz.BackupAccount, 0)
	var total int64
	err := r.db.Model(&biz.BackupAccount{}).Order("id desc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&accounts).Error
	return accounts, total, err
}

func (r backupAccountRepo) Get(id uint) (*biz.BackupAccount, error) {
	account := new(biz.BackupAccount)
	err := r.db.Model(&biz.BackupAccount{}).Where("id = ?", id).First(account).Error
	return account, err
}

func (r backupAccountRepo) Create(ctx context.Context, req *request.BackupAccountCreate) (*biz.BackupAccount, error) {
	account := &biz.BackupAccount{
		Type: biz.BackupAccountType(req.Type),
		Name: req.Name,
		Info: req.Info,
	}

	if err := r.db.Create(account).Error; err != nil {
		return nil, err
	}

	r.log.Info("backup account created", slog.String("type", biz.OperationTypeBackup), slog.Uint64("operator_id", getOperatorID(ctx)), slog.Uint64("id", uint64(account.ID)), slog.String("account_type", req.Type), slog.String("name", req.Name))

	return account, nil
}

func (r backupAccountRepo) Update(ctx context.Context, req *request.BackupAccountUpdate) error {
	account, err := r.Get(req.ID)
	if err != nil {
		return err
	}

	account.Type = biz.BackupAccountType(req.Type)
	account.Name = req.Name
	account.Info = req.Info

	if err = r.db.Save(account).Error; err != nil {
		return err
	}

	r.log.Info("backup account updated", slog.String("type", biz.OperationTypeBackup), slog.Uint64("operator_id", getOperatorID(ctx)), slog.Uint64("id", uint64(req.ID)), slog.String("account_type", req.Type))

	return nil
}

func (r backupAccountRepo) Delete(ctx context.Context, id uint) error {
	if err := r.db.Model(&biz.BackupAccount{}).Where("id = ?", id).Delete(&biz.BackupAccount{}).Error; err != nil {
		return err
	}

	r.log.Info("backup account deleted", slog.String("type", biz.OperationTypeBackup), slog.Uint64("operator_id", getOperatorID(ctx)), slog.Uint64("id", uint64(id)))

	return nil
}
