package bootstrap

import (
	"log/slog"
	"path/filepath"

	"github.com/DeRuina/timberjack"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/pkg/config"
)

func NewLog(conf *config.Config) *slog.Logger {
	tjLogger := &timberjack.Logger{
		Filename:    filepath.Join(app.Root, "panel/storage/logs/app.log"),
		MaxSize:     10,
		MaxAge:      30,
		Compression: "zstd",
	}

	level := slog.LevelInfo
	if conf.App.Debug {
		level = slog.LevelDebug
	}

	log := slog.New(slog.NewJSONHandler(tjLogger, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(log)

	return log
}
