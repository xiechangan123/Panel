package bootstrap

import (
	"log/slog"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/taskqueue"
	"github.com/acepanel/panel/v3/pkg/types"
)

// NewRunner 创建任务运行器
func NewRunner(i do.Injector) (types.TaskRunner, error) {
	return taskqueue.NewRunner(
		do.MustInvoke[*gorm.DB](i),
		do.MustInvoke[*slog.Logger](i),
		do.MustInvoke[*biz.NotifyUsecase](i),
		do.MustInvoke[*gotext.Locale](i),
	), nil
}
