package bootstrap

import (
	"log/slog"

	"github.com/leonelquinteros/gotext"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/taskqueue"
	"github.com/acepanel/panel/v3/pkg/types"
)

// NewRunner 创建任务运行器
func NewRunner(notifyUsecase *biz.NotifyUsecase, db *gorm.DB, t *gotext.Locale, log *slog.Logger) types.TaskRunner {
	return taskqueue.NewRunner(db, log, notifyUsecase, t)
}
