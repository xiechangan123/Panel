package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// CutoffCommand 日志切割命令组
func CutoffCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "cutoff",
		Usage: t.Get("Log rotation"),
		Commands: []*cli.Command{
			{
				Name:  "website",
				Usage: t.Get("Website"),
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
					return do.MustInvoke[*service.CliService](i).CutoffWebsite(ctx, cmd)
				},
			},
			{
				Name:  "container",
				Usage: t.Get("Container"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    t.Get("Container name"),
						Required: true,
					},
					&cli.UintFlag{
						Name:    "storage",
						Aliases: []string{"s"},
						Usage:   t.Get("Storage ID (local storage if not filled)"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).CutoffContainer(ctx, cmd)
				},
			},
			{
				Name:  "clear",
				Usage: t.Get("Clear rotated logs"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "type",
						Aliases:  []string{"t"},
						Usage:    t.Get("Rotation type"),
						Required: true,
					},
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    t.Get("Target name"),
						Required: true,
					},
					&cli.UintFlag{
						Name:     "keep",
						Aliases:  []string{"k"},
						Usage:    t.Get("Number of logs to keep"),
						Required: true,
					},
					&cli.UintFlag{
						Name:    "storage",
						Aliases: []string{"s"},
						Usage:   t.Get("Storage ID (local storage if not filled)"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).CutoffClear(ctx, cmd)
				},
			},
		},
	}, nil
}
