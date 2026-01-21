package data

import (
	"context"
	"log/slog"

	"github.com/leonelquinteros/gotext"
	"gorm.io/gorm"

	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/types"
)

type backupAccountRepo struct {
	t       *gotext.Locale
	db      *gorm.DB
	log     *slog.Logger
	setting biz.SettingRepo
}

func NewBackupAccountRepo(t *gotext.Locale, db *gorm.DB, log *slog.Logger, setting biz.SettingRepo) biz.BackupAccountRepo {
	return &backupAccountRepo{
		t:       t,
		db:      db,
		log:     log,
		setting: setting,
	}
}

func (r backupAccountRepo) List(page, limit uint) ([]*biz.BackupAccount, int64, error) {
	// 本地存储
	localStorage, err := r.Get(0)
	if err != nil {
		return nil, 0, err
	}

	var dbAccounts []*biz.BackupAccount
	var total int64
	if err = r.db.Model(&biz.BackupAccount{}).Order("id asc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&dbAccounts).Error; err != nil {
		return nil, 0, err
	}

	accounts := make([]*biz.BackupAccount, 0, len(dbAccounts)+1)
	if page == 1 {
		accounts = append(accounts, localStorage)
	}
	accounts = append(accounts, dbAccounts...)

	return accounts, total + 1, nil
}

func (r backupAccountRepo) Get(id uint) (*biz.BackupAccount, error) {
	if id == 0 {
		path, err := r.setting.Get(biz.SettingKeyBackupPath)
		if err != nil {
			return nil, err
		}
		return &biz.BackupAccount{
			ID:   0,
			Type: biz.BackupAccountTypeLocal,
			Name: r.t.Get("Local Storage"),
			Info: types.BackupAccountInfo{
				Path: path,
			},
		}, nil
	}

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
