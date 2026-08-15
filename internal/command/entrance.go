package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// EntranceCommand 访问入口管理命令组
func EntranceCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "entrance",
		Usage: t.Get("Operate AcePanel access entrance"),
		Commands: []*cli.Command{
			{
				Name:  "on",
				Usage: t.Get("Enable access entrance"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.EntranceOn(ctx, cmd)
				},
			},
			{
				Name:  "off",
				Usage: t.Get("Disable access entrance"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.EntranceOff(ctx, cmd)
				},
			},
		},
	}
}

// BindDomainCommand 域名绑定管理命令组
func BindDomainCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "bind-domain",
		Usage: t.Get("Operate AcePanel domain binding"),
		Commands: []*cli.Command{
			{
				Name:      "on",
				Usage:     t.Get("Enable domain binding"),
				ArgsUsage: t.Get("<domain> [domain...]"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.BindDomainOn(ctx, cmd)
				},
			},
			{
				Name:  "off",
				Usage: t.Get("Disable domain binding"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.BindDomainOff(ctx, cmd)
				},
			},
		},
	}
}

// BindIPCommand IP 绑定管理命令组
func BindIPCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "bind-ip",
		Usage: t.Get("Operate AcePanel IP binding"),
		Commands: []*cli.Command{
			{
				Name:      "on",
				Usage:     t.Get("Enable IP binding"),
				ArgsUsage: t.Get("<ip> [ip...]"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.BindIPOn(ctx, cmd)
				},
			},
			{
				Name:  "off",
				Usage: t.Get("Disable IP binding"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.BindIPOff(ctx, cmd)
				},
			},
		},
	}
}

// BindUACommand UA 绑定管理命令组
func BindUACommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "bind-ua",
		Usage: t.Get("Operate AcePanel UA binding"),
		Commands: []*cli.Command{
			{
				Name:      "on",
				Usage:     t.Get("Enable UA binding"),
				ArgsUsage: t.Get("<user-agent> [user-agent...]"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.BindUAOn(ctx, cmd)
				},
			},
			{
				Name:  "off",
				Usage: t.Get("Disable UA binding"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.BindUAOff(ctx, cmd)
				},
			},
		},
	}
}
