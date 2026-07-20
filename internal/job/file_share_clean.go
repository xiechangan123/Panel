package job

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
)

// FileShareClean 过期文件分享清理任务
type FileShareClean struct {
	log           *slog.Logger
	fileShareRepo *biz.FileShareUsecase
}

// NewFileShareClean 构造过期文件分享清理任务
func NewFileShareClean(i do.Injector) (Job, error) {
	return Job{
		Spec: "0 * * * *",
		Task: &FileShareClean{
			log:           do.MustInvoke[*slog.Logger](i),
			fileShareRepo: do.MustInvoke[*biz.FileShareUsecase](i),
		},
	}, nil
}

func (r *FileShareClean) Run(_ context.Context) error {
	if app.Status != app.StatusNormal {
		return nil
	}

	count, err := r.fileShareRepo.ClearExpired()
	if err != nil {
		r.log.Warn("failed to clear expired file shares", slog.Any("err", err))
		return nil
	}
	if count > 0 {
		r.log.Info("expired file shares cleared", slog.Int64("count", count))
	}
	return nil
}
