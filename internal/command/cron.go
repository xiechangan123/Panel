package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// CronCommand 计划任务命令组
func CronCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "cron",
		Usage: t.Get("Cron task"),
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: t.Get("List all cron tasks"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.CronList(ctx, cmd)
				},
			},
			{
				Name:  "run",
				Usage: t.Get("Run a cron task immediately"),
				Flags: []cli.Flag{
					&cli.UintFlag{
						Name:     "id",
						Aliases:  []string{"i"},
						Usage:    t.Get("Cron task ID"),
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.CronRun(ctx, cmd)
				},
			},
			{
				Name:  "status",
				Usage: t.Get("Enable or disable a cron task"),
				Flags: []cli.Flag{
					&cli.UintFlag{
						Name:     "id",
						Aliases:  []string{"i"},
						Usage:    t.Get("Cron task ID"),
						Required: true,
					},
					&cli.BoolFlag{
						Name:  "off",
						Usage: t.Get("Disable the task instead of enabling it"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.CronStatus(ctx, cmd)
				},
			},
			{
				Name:  "failed",
				Usage: t.Get("Report a failed cron task, called by the task wrapper script"),
				Flags: []cli.Flag{
					&cli.UintFlag{
						Name:     "id",
						Aliases:  []string{"i"},
						Usage:    t.Get("Cron task ID"),
						Required: true,
					},
					&cli.IntFlag{
						Name:    "code",
						Aliases: []string{"c"},
						Usage:   t.Get("Exit code"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.CronFailed(ctx, cmd)
				},
			},
		},
	}
}
