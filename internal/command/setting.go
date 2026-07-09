package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// SettingCommand 面板设置管理命令组
func SettingCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:   "setting",
		Usage:  t.Get("Setting management"),
		Hidden: true,
		Commands: []*cli.Command{
			{
				Name:   "get",
				Usage:  t.Get("Get panel setting (use only under guidance)"),
				Hidden: true,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).GetSetting(ctx, cmd)
				},
			},
			{
				Name:   "write",
				Usage:  t.Get("Write panel setting (use only under guidance)"),
				Hidden: true,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).WriteSetting(ctx, cmd)
				},
			},
			{
				Name:   "remove",
				Usage:  t.Get("Remove panel setting (use only under guidance)"),
				Hidden: true,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).RemoveSetting(ctx, cmd)
				},
			},
		},
	}, nil
}
