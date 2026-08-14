package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/acepanel/panel/v3/internal/biz"
)

var TamperMigrations = []*gormigrate.Migration{
	{
		ID: "20260720-init-tamper-logs",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&biz.TamperLog{})
		},
	},
	{
		ID: "20260814-compress-tamper-logs",
		Migrate: func(tx *gorm.DB) error {
			if tx.Migrator().HasIndex(&biz.TamperLog{}, "idx_tamper_logs_path") {
				if err := tx.Migrator().DropIndex(&biz.TamperLog{}, "idx_tamper_logs_path"); err != nil {
					return err
				}
			}

			for {
				var items []*biz.TamperLog
				if err := tx.Where("typeof(path) = 'text'").Order("id").Limit(batchSize).Find(&items).Error; err != nil {
					return err
				}
				if len(items) == 0 {
					break
				}
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "id"}},
					DoUpdates: clause.AssignmentColumns([]string{"path"}),
				}).Create(&items).Error; err != nil {
					return err
				}
			}

			return vacuumDB(tx)
		},
	},
}
