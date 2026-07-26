package bootstrap

import (
	"log/slog"
	"path/filepath"
	"time"

	"github.com/libtnb/logrotate"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/pkg/config"
)

type Logger struct {
	*slog.Logger
}

// NewLogger 构建写入轮转文件的应用日志。
func NewLogger(conf *config.Config) (*Logger, func(), error) {
	w, err := logrotate.New(filepath.Join(app.Root, "panel/storage/logs/app.log"),
		logrotate.WithMaxSize(10*logrotate.MB),
		logrotate.WithMaxAge(30*logrotate.Day),
		logrotate.WithRotateAt("00:00"),
		logrotate.WithFileMode(0o600),
		logrotate.WithLocation(time.Local),
	)
	if err != nil {
		return nil, nil, err
	}

	level := slog.LevelInfo
	if conf.App.Debug {
		level = slog.LevelDebug
	}

	log := slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(log)

	cleanup := func() { _ = w.Close() }
	return &Logger{Logger: log}, cleanup, nil
}

// NewSlog 解包出纯 *slog.Logger 供应用其余部分使用。
func NewSlog(logger *Logger) *slog.Logger {
	return logger.Logger
}
