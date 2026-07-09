package bootstrap

import (
	"log/slog"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/taskqueue"
	"github.com/acepanel/panel/v3/pkg/types"
)

// NewRunner 创建任务运行器
func NewRunner(i do.Injector) (types.TaskRunner, error) {
	return taskqueue.NewRunner(do.MustInvoke[*gorm.DB](i), do.MustInvoke[*slog.Logger](i)), nil
}
