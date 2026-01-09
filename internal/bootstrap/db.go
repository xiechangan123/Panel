package bootstrap

import (
	"log/slog"
	"path/filepath"

	"github.com/DeRuina/timberjack"
	"github.com/go-gormigrate/gormigrate/v2"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	sloggorm "github.com/orandin/slog-gorm"
	"gorm.io/gorm"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/migration"
	"github.com/acepanel/panel/pkg/config"
)

func NewDB(conf *config.Config) (*gorm.DB, error) {
	tjLogger := &timberjack.Logger{
		Filename:    filepath.Join(app.Root, "panel/storage/logs/db.log"),
		MaxSize:     10,
		MaxAge:      30,
		Compression: "zstd",
	}

	handler := slog.New(slog.NewJSONHandler(tjLogger, nil)).Handler()
	options := []sloggorm.Option{sloggorm.WithHandler(handler)}
	if conf.Database.Debug {
		options = append(options, sloggorm.WithTraceAll())
	}

	return gorm.Open(gormlite.Open("file:"+filepath.Join(app.Root, "panel/storage/panel.db?_txlock=immediate")), &gorm.Config{
		Logger:                                   sloggorm.New(options...),
		SkipDefaultTransaction:                   true,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
}

func NewMigrate(db *gorm.DB) *gormigrate.Gormigrate {
	return gormigrate.New(db, &gormigrate.Options{
		UseTransaction: true, // Note: MySQL not support DDL transaction
	}, migration.Migrations)
}
