package bootstrap

import (
	"log/slog"

	"gorm.io/gorm"

	"github.com/acepanel/panel/internal/taskqueue"
	"github.com/acepanel/panel/pkg/types"
)

// NewRunner 创建任务运行器
func NewRunner(db *gorm.DB, log *slog.Logger) types.TaskRunner {
	return taskqueue.NewRunner(db, log)
}
