package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// SettingCommand 面板设置管理命令组
func SettingCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:   "setting",
		Usage:  t.Get("Setting management"),
		Hidden: true,
		Commands: []*cli.Command{
			{
				Name:      "get",
				Usage:     t.Get("Get panel setting (use only under guidance)"),
				ArgsUsage: t.Get("<key>"),
				Hidden:    true,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.GetSetting(ctx, cmd)
				},
			},
			{
				Name:      "write",
				Usage:     t.Get("Write panel setting (use only under guidance)"),
				ArgsUsage: t.Get("<key> <value>"),
				Hidden:    true,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.WriteSetting(ctx, cmd)
				},
			},
			{
				Name:      "remove",
				Usage:     t.Get("Remove panel setting (use only under guidance)"),
				ArgsUsage: t.Get("<key>"),
				Hidden:    true,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.RemoveSetting(ctx, cmd)
				},
			},
		},
	}
}
