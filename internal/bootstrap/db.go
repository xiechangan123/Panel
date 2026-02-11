package bootstrap

import (
	"log/slog"
	"path/filepath"

	"github.com/DeRuina/timberjack"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/libtnb/sqlite"
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
		LocalTime:   true,
		RotateAt:    []string{"00:00"},
		FileMode:    0o600,
		Compression: "none",
	}

	handler := slog.New(slog.NewJSONHandler(tjLogger, nil)).Handler()
	options := []sloggorm.Option{sloggorm.WithHandler(handler)}
	if conf.Database.Debug {
		options = append(options, sloggorm.WithTraceAll())
	}

	db, err := gorm.Open(sqlite.Open(filepath.Join(app.Root, "panel/storage/panel.db?_txlock=immediate&_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)")),
		&gorm.Config{
			Logger:                                   sloggorm.New(options...),
			SkipDefaultTransaction:                   true,
			DisableForeignKeyConstraintWhenMigrating: true,
		})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	return db, nil
}

func NewMigrate(db *gorm.DB) *gormigrate.Gormigrate {
	return gormigrate.New(db, &gormigrate.Options{
		UseTransaction: true, // Note: MySQL not support DDL transaction
	}, migration.Migrations)
}
