package bootstrap

import (
	"log/slog"
	"path/filepath"
	"time"

	"github.com/libtnb/logrotate"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/pkg/config"
)

type Logger struct {
	*slog.Logger
	close func() error
}

func (l *Logger) Shutdown() error {
	return l.close()
}

// NewLogger 构建写入轮转文件的应用日志。
func NewLogger(i do.Injector) (*Logger, error) {
	conf := do.MustInvoke[*config.Config](i)

	w, err := logrotate.New(filepath.Join(app.Root, "panel/storage/logs/app.log"),
		logrotate.WithMaxSize(10*logrotate.MB),
		logrotate.WithMaxAge(30*logrotate.Day),
		logrotate.WithRotateAt("00:00"),
		logrotate.WithFileMode(0o600),
		logrotate.WithLocation(time.Local),
	)
	if err != nil {
		return nil, err
	}

	level := slog.LevelInfo
	if conf.App.Debug {
		level = slog.LevelDebug
	}

	log := slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(log)

	return &Logger{Logger: log, close: w.Close}, nil
}

// NewSlog 解包出纯 *slog.Logger 供应用其余部分使用。
func NewSlog(i do.Injector) (*slog.Logger, error) {
	return do.MustInvoke[*Logger](i).Logger, nil
}
