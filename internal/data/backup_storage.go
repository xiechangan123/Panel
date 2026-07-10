package data

import (
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
)

type backupAccountRepo struct {
	db *gorm.DB
}

func NewBackupAccountRepo(i do.Injector) (biz.BackupAccountRepo, error) {
	return &backupAccountRepo{
		db: do.MustInvoke[*gorm.DB](i),
	}, nil
}

func (r backupAccountRepo) ListPaged(page, limit uint) ([]*biz.BackupStorage, int64, error) {
	var dbAccounts []*biz.BackupStorage
	var total int64
	if err := r.db.Model(&biz.BackupStorage{}).Order("id asc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&dbAccounts).Error; err != nil {
		return nil, 0, err
	}

	return dbAccounts, total, nil
}

func (r backupAccountRepo) GetByID(id uint) (*biz.BackupStorage, error) {
	account := new(biz.BackupStorage)
	err := r.db.Model(&biz.BackupStorage{}).Where("id = ?", id).First(account).Error
	return account, err
}

func (r backupAccountRepo) Create(account *biz.BackupStorage) error {
	return r.db.Create(account).Error
}

func (r backupAccountRepo) Update(account *biz.BackupStorage) error {
	return r.db.Save(account).Error
}

func (r backupAccountRepo) Delete(id uint) error {
	return r.db.Model(&biz.BackupStorage{}).Where("id = ?", id).Delete(&biz.BackupStorage{}).Error
}
