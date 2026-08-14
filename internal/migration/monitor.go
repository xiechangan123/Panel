package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
)

var MonitorMigrations = []*gormigrate.Migration{
	{
		ID: "20260814-init-monitor",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&biz.Monitor{})
		},
	},
}
