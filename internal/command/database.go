package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// DatabaseCommand 数据库管理命令组
func DatabaseCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "database",
		Usage: t.Get("Database management"),
		Commands: []*cli.Command{
			{
				Name:  "add-server",
				Usage: t.Get("Add database server"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "type",
						Usage:    t.Get("Server type"),
						Required: true,
					},
					&cli.StringFlag{
						Name:     "name",
						Usage:    t.Get("Server name"),
						Required: true,
					},
					&cli.StringFlag{
						Name:     "host",
						Usage:    t.Get("Server address"),
						Required: true,
					},
					&cli.UintFlag{
						Name:     "port",
						Usage:    t.Get("Server port"),
						Required: true,
					},
					&cli.StringFlag{
						Name:  "username",
						Usage: t.Get("Server username"),
					},
					&cli.StringFlag{
						Name:  "password",
						Usage: t.Get("Server password"),
					},
					&cli.StringFlag{
						Name:  "remark",
						Usage: t.Get("Server remark"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).DatabaseAddServer(ctx, cmd)
				},
			},
			{
				Name:  "delete-server",
				Usage: t.Get("Delete database server"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Usage:    t.Get("Server name"),
						Aliases:  []string{"n"},
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).DatabaseDeleteServer(ctx, cmd)
				},
			},
		},
	}, nil
}
