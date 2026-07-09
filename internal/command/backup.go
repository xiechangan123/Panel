package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// BackupCommand 数据备份命令组
func BackupCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "backup",
		Usage: t.Get("Data backup"),
		Commands: []*cli.Command{
			{
				Name:  "website",
				Usage: t.Get("Backup website"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    t.Get("Website name"),
						Required: true,
					},
					&cli.UintFlag{
						Name:    "storage",
						Aliases: []string{"s"},
						Usage:   t.Get("Storage ID (local storage if not filled)"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).BackupWebsite(ctx, cmd)
				},
			},
			{
				Name:  "database",
				Usage: t.Get("Backup database"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "type",
						Aliases:  []string{"t"},
						Usage:    t.Get("Database type"),
						Required: true,
					},
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    t.Get("Database name"),
						Required: true,
					},
					&cli.UintFlag{
						Name:    "storage",
						Aliases: []string{"s"},
						Usage:   t.Get("Storage ID (local storage if not filled)"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).BackupDatabase(ctx, cmd)
				},
			},
			{
				Name:  "path",
				Usage: t.Get("Backup directory"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "path",
						Aliases:  []string{"p"},
						Usage:    t.Get("Directory path"),
						Required: true,
					},
					&cli.UintFlag{
						Name:    "storage",
						Aliases: []string{"s"},
						Usage:   t.Get("Storage ID (local storage if not filled)"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).BackupPath(ctx, cmd)
				},
			},
			{
				Name:  "panel",
				Usage: t.Get("Backup panel"),
				Flags: []cli.Flag{},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).BackupPanel(ctx, cmd)
				},
			},
			{
				Name:  "clear",
				Usage: t.Get("Clear backups"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "type",
						Aliases:  []string{"t"},
						Usage:    t.Get("Backup type"),
						Required: true,
					},
					&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    t.Get("Backup file"),
						Required: true,
					},
					&cli.UintFlag{
						Name:     "keep",
						Aliases:  []string{"k"},
						Usage:    t.Get("Number of backups to keep"),
						Required: true,
					},
					&cli.UintFlag{
						Name:    "storage",
						Aliases: []string{"s"},
						Usage:   t.Get("Storage ID (local storage if not filled)"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).BackupClear(ctx, cmd)
				},
			},
		},
	}, nil
}
