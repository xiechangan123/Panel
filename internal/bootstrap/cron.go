package bootstrap

import (
	"log/slog"

	"github.com/libtnb/cron"
	"github.com/libtnb/cron/wrap"

	"github.com/acepanel/panel/v3/internal/job"
)

func NewCron(log *slog.Logger, jobs *job.Jobs) (*cron.Cron, error) {
	c := cron.New(
		cron.WithLogger(log),
		cron.WithChain(wrap.Recover(), wrap.SkipIfRunning()),
	)
	if err := jobs.Register(c); err != nil {
		return nil, err
	}

	return c, nil
}
