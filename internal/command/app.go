package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// AppCommand 应用管理命令组
func AppCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "app",
		Usage: t.Get("Application management"),
		Commands: []*cli.Command{
			{
				Name:      "install",
				Usage:     t.Get("Install application"),
				ArgsUsage: t.Get("<slug> [channel]"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.AppInstall(ctx, cmd)
				},
			},
			{
				Name:      "uninstall",
				Usage:     t.Get("Uninstall application"),
				ArgsUsage: t.Get("<slug>"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.AppUnInstall(ctx, cmd)
				},
			},
			{
				Name:      "update",
				Usage:     t.Get("Update application"),
				ArgsUsage: t.Get("<slug>"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.AppUpdate(ctx, cmd)
				},
			},
			{
				Name:  "list",
				Usage: t.Get("List installed applications"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.AppList(ctx, cmd)
				},
			},
			{
				Name:      "write",
				Usage:     t.Get("Add panel application mark (use only under guidance)"),
				ArgsUsage: t.Get("<slug> <channel> <version>"),
				Hidden:    true,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.AppWrite(ctx, cmd)
				},
			},
			{
				Name:      "remove",
				Usage:     t.Get("Remove panel application mark (use only under guidance)"),
				ArgsUsage: t.Get("<slug>"),
				Hidden:    true,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.AppRemove(ctx, cmd)
				},
			},
		},
	}
}
