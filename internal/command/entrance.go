package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// EntranceCommand 访问入口管理命令组
func EntranceCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "entrance",
		Usage: t.Get("Operate AcePanel access entrance"),
		Commands: []*cli.Command{
			{
				Name:  "on",
				Usage: t.Get("Enable access entrance"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).EntranceOn(ctx, cmd)
				},
			},
			{
				Name:  "off",
				Usage: t.Get("Disable access entrance"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).EntranceOff(ctx, cmd)
				},
			},
		},
	}, nil
}

// BindDomainCommand 域名绑定管理命令组
func BindDomainCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "bind-domain",
		Usage: t.Get("Operate AcePanel domain binding"),
		Commands: []*cli.Command{
			{
				Name:  "off",
				Usage: t.Get("Disable domain binding"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).BindDomainOff(ctx, cmd)
				},
			},
		},
	}, nil
}

// BindIPCommand IP 绑定管理命令组
func BindIPCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "bind-ip",
		Usage: t.Get("Operate AcePanel IP binding"),
		Commands: []*cli.Command{
			{
				Name:  "off",
				Usage: t.Get("Disable IP binding"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).BindIPOff(ctx, cmd)
				},
			},
		},
	}, nil
}

// BindUACommand UA 绑定管理命令组
func BindUACommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "bind-ua",
		Usage: t.Get("Operate AcePanel UA binding"),
		Commands: []*cli.Command{
			{
				Name:  "off",
				Usage: t.Get("Disable UA binding"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).BindUAOff(ctx, cmd)
				},
			},
		},
	}, nil
}
