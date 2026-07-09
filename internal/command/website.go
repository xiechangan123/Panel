package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// WebsiteCommand 网站管理命令组
func WebsiteCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "website",
		Usage: t.Get("Website management"),
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: t.Get("Create new website"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Usage:    t.Get("Website name"),
						Aliases:  []string{"n"},
						Required: true,
					},
					&cli.StringSliceFlag{
						Name:     "domains",
						Usage:    t.Get("List of domains associated with the website"),
						Aliases:  []string{"d"},
						Required: true,
					},
					&cli.StringSliceFlag{
						Name:     "listens",
						Usage:    t.Get("List of listening addresses associated with the website"),
						Aliases:  []string{"l"},
						Required: true,
					},
					&cli.StringFlag{
						Name:  "path",
						Usage: t.Get("Path where the website is hosted (default path if not filled)"),
					},
					&cli.UintFlag{
						Name:  "php",
						Usage: t.Get("PHP version used by the website (not used if not filled)"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).WebsiteCreate(ctx, cmd)
				},
			},
			{
				Name:  "remove",
				Usage: t.Get("Remove website"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Usage:    t.Get("Website name"),
						Aliases:  []string{"n"},
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).WebsiteRemove(ctx, cmd)
				},
			},
			{
				Name:  "delete",
				Usage: t.Get("Delete website (including website directory, database with the same name)"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Usage:    t.Get("Website name"),
						Aliases:  []string{"n"},
						Required: true,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).WebsiteDelete(ctx, cmd)
				},
			},
			{
				Name:   "write",
				Usage:  t.Get("Write website data (use only under guidance)"),
				Hidden: true,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).WebsiteWrite(ctx, cmd)
				},
			},
		},
	}, nil
}
