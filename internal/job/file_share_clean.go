package job

import (
	"context"
	"log/slog"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
)

// FileShareClean 过期文件分享清理任务
type FileShareClean struct {
	log           *slog.Logger
	fileShareRepo *biz.FileShareUsecase
}

// NewFileShareClean 构造过期文件分享清理任务
func NewFileShareClean(fileShareUsecase *biz.FileShareUsecase, log *slog.Logger) Job {
	return Job{
		Spec: "0 * * * *",
		Task: &FileShareClean{
			log:           log,
			fileShareRepo: fileShareUsecase,
		},
	}
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
