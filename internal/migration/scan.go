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
			if tx.Migrator().HasTable("scan_events") {
				return nil
			}
			return tx.AutoMigrate(&biz.ScanSource{}, &biz.ScanEvent{})
		},
	},
	{
		ID: "20260814-normalize-scan-events",
		Migrate: func(tx *gorm.DB) error {
			if !tx.Migrator().HasColumn("scan_events", "source_ip") {
				return nil
			}

			return tx.Transaction(func(tx *gorm.DB) error {
				if err := tx.Migrator().RenameTable("scan_events", "scan_events_legacy"); err != nil {
					return err
				}
				if err := tx.Exec("DROP INDEX IF EXISTS idx_scan_unique").Error; err != nil {
					return err
				}
				if err := tx.Exec("DROP INDEX IF EXISTS idx_scan_date").Error; err != nil {
					return err
				}
				if err := tx.AutoMigrate(&biz.ScanSource{}, &biz.ScanEvent{}); err != nil {
					return err
				}

				if err := tx.Exec(`
					INSERT INTO scan_sources (source_ip, country, region, city, isp)
					SELECT source_ip, MAX(country), MAX(region), MAX(city), MAX(isp)
					FROM scan_events_legacy
					WHERE 1 = 1
					GROUP BY source_ip
					ON CONFLICT(source_ip) DO UPDATE SET
						country = excluded.country,
						region = excluded.region,
						city = excluded.city,
						isp = excluded.isp
				`).Error; err != nil {
					return err
				}
				if err := tx.Exec(`
					INSERT INTO scan_events (id, source_id, port, protocol, date, count, first_seen, last_seen)
					SELECT events.id, sources.id, events.port, events.protocol, events.date,
						events.count, events.first_seen, events.last_seen
					FROM scan_events_legacy AS events
					JOIN scan_sources AS sources ON sources.source_ip = events.source_ip
				`).Error; err != nil {
					return err
				}

				return tx.Migrator().DropTable("scan_events_legacy")
			})
		},
	},
}
