package bootstrap

import (
	"log/slog"

	"github.com/robfig/cron/v3"

	"github.com/acepanel/panel/internal/job"
	"github.com/acepanel/panel/pkg/config"
	pkgcron "github.com/acepanel/panel/pkg/cron"
)

func NewCron(conf *config.Config, log *slog.Logger, jobs *job.Jobs) (*cron.Cron, error) {
	logger := pkgcron.NewLogger(log, conf.App.Debug)

	c := cron.New(
		cron.WithParser(cron.NewParser(
			cron.SecondOptional|cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor,
		)),
		cron.WithLogger(logger),
		cron.WithChain(cron.Recover(logger), cron.SkipIfStillRunning(logger)),
	)
	if err := jobs.Register(c); err != nil {
		return nil, err
	}

	return c, nil
}
