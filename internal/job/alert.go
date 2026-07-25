package job

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
)

// Alert 告警规则评估任务
type Alert struct {
	log       *slog.Logger
	alertRepo *biz.AlertUsecase
}

// NewAlert 构造告警评估任务
func NewAlert(i do.Injector) (Job, error) {
	return Job{
		Spec: "* * * * *",
		Task: &Alert{
			log:       do.MustInvoke[*slog.Logger](i),
			alertRepo: do.MustInvoke[*biz.AlertUsecase](i),
		},
	}, nil
}

func (r *Alert) Run(ctx context.Context) error {
	if app.Status != app.StatusNormal {
		return nil
	}

	if err := r.alertRepo.Evaluate(ctx); err != nil {
		r.log.Warn("failed to evaluate alert rules", slog.Any("err", err))
	}

	return nil
}
