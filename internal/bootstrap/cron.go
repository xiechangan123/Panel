package bootstrap

import (
	"log/slog"

	"github.com/libtnb/cron"
	"github.com/libtnb/cron/wrap"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/job"
)

func NewCron(i do.Injector) (*cron.Cron, error) {
	// 面板任务均为 5 段表达式，不启用 WithSecondsField
	c := cron.New(
		cron.WithLogger(do.MustInvoke[*slog.Logger](i)),
		cron.WithChain(wrap.Recover(), wrap.SkipIfRunning()),
	)
	if err := job.Register(i, c); err != nil {
		return nil, err
	}

	return c, nil
}
