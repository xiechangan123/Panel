package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
)

var ScanMigrations = []*gormigrate.Migration{
	{
		ID: "20260218-init-scan-events",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&biz.ScanEvent{})
		},
	},
}
