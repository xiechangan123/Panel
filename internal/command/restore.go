package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// RestoreCommand 数据恢复命令组
func RestoreCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "restore",
		Usage: t.Get("Data restore"),
		Commands: []*cli.Command{
			{
				Name:  "website",
				Usage: t.Get("Restore website backup"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    t.Get("Website name"),
						Required: true,
					},
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    t.Get("Backup file (absolute path or filename under default backup path)"),
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.RestoreWebsite(ctx, cmd)
				},
			},
			{
				Name:  "database",
				Usage: t.Get("Restore database backup"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "type",
						Aliases:  []string{"t"},
						Usage:    t.Get("Database type (mysql, postgresql, clickhouse, redis, valkey)"),
						Required: true,
					},
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    t.Get("Database name"),
						Required: true,
					},
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    t.Get("Backup file (absolute path or filename under default backup path)"),
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.RestoreDatabase(ctx, cmd)
				},
			},
			{
				Name:  "panel",
				Usage: t.Get("Restore panel backup (the panel restarts automatically after restoring)"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    t.Get("Backup file (absolute path or filename under default backup path)"),
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.RestorePanel(ctx, cmd)
				},
			},
		},
	}
}
