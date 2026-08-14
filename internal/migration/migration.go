package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

var Migrations []*gormigrate.Migration

const batchSize = 100

func vacuumDB(db *gorm.DB) error {
	if err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error; err != nil {
		return err
	}
	if err := db.Exec("VACUUM").Error; err != nil {
		return err
	}
	return db.Exec("PRAGMA optimize").Error
}
