package bootstrap

import (
	"log/slog"
	"path/filepath"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/knadh/koanf/v2"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	sloggorm "github.com/orandin/slog-gorm"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/gorm"

	"github.com/tnborg/panel/internal/app"
	"github.com/tnborg/panel/internal/migration"
)

func NewDB(conf *koanf.Koanf) (*gorm.DB, error) {
	ljLogger := &lumberjack.Logger{
		Filename: filepath.Join(app.Root, "panel/storage/logs/db.log"),
		MaxSize:  10,
		MaxAge:   30,
		Compress: true,
	}

	handler := slog.New(slog.NewJSONHandler(ljLogger, nil)).Handler()
	options := []sloggorm.Option{sloggorm.WithHandler(handler)}
	if conf.Bool("database.debug") {
		options = append(options, sloggorm.WithTraceAll())
	}

	return gorm.Open(gormlite.Open("file:"+filepath.Join(app.Root, "panel/storage/app.db?_txlock=immediate")), &gorm.Config{
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
