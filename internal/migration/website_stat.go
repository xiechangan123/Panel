package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/acepanel/panel/v3/internal/biz"
)

var WebsiteStatMigrations = []*gormigrate.Migration{
	{
		ID: "20260814-init-website-stat",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(
				&biz.WebsiteStat{}, &biz.WebsiteErrorLog{},
				&biz.WebsiteStatSpider{}, &biz.WebsiteStatClient{},
				&biz.WebsiteStatIP{}, &biz.WebsiteStatURI{},
			)
		},
	},
	{
		ID: "20260814-compress-website-error-logs",
		Migrate: func(tx *gorm.DB) error {
			for {
				var items []*biz.WebsiteErrorLog
				if err := tx.Where("typeof(uri) = 'text' OR typeof(ua) = 'text' OR typeof(body) = 'text'").
					Order("id").Limit(batchSize).Find(&items).Error; err != nil {
					return err
				}
				if len(items) == 0 {
					break
				}
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "id"}},
					DoUpdates: clause.AssignmentColumns([]string{"uri", "ua", "body"}),
				}).Create(&items).Error; err != nil {
					return err
				}
			}

			return vacuumDB(tx)
		},
	},
}
