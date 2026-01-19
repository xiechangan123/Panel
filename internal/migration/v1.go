package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/internal/biz"
)

func init() {
	Migrations = append(Migrations, &gormigrate.Migration{
		ID: "20260101-init",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(
				&biz.App{},
				&biz.Cache{},
				&biz.Cert{},
				&biz.CertAccount{},
				&biz.CertDNS{},
				&biz.Cron{},
				&biz.DatabaseServer{},
				&biz.DatabaseUser{},
				&biz.Monitor{},
				&biz.Setting{},
				&biz.SSH{},
				&biz.Task{},
				&biz.User{},
				&biz.UserToken{},
				&biz.WebHook{},
				&biz.Website{},
			)
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(
				&biz.App{},
				&biz.Cert{},
				&biz.CertAccount{},
				&biz.CertDNS{},
				&biz.Cron{},
				&biz.DatabaseServer{},
				&biz.DatabaseUser{},
				&biz.Monitor{},
				&biz.Setting{},
				&biz.SSH{},
				&biz.Task{},
				&biz.User{},
				&biz.UserToken{},
				&biz.WebHook{},
				&biz.Website{},
			)
		},
	})
	Migrations = append(Migrations, &gormigrate.Migration{
		ID: "20260110-add-project",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&biz.Project{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&biz.Project{})
		},
	})
	Migrations = append(Migrations, &gormigrate.Migration{
		ID: "20260120-add-backup",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(
				&biz.Backup{},
				&biz.BackupAccount{},
			)
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(
				&biz.Backup{},
				&biz.BackupAccount{},
			)
		},
	})
}
