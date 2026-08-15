package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// FirewallCommand 防火墙管理命令组
func FirewallCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "firewall",
		Usage: t.Get("Firewall management"),
		Commands: []*cli.Command{
			{
				Name:  "status",
				Usage: t.Get("Get firewall status"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.FirewallStatus(ctx, cmd)
				},
			},
			{
				Name:  "on",
				Usage: t.Get("Enable firewall"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.FirewallOn(ctx, cmd)
				},
			},
			{
				Name:  "off",
				Usage: t.Get("Disable firewall"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.FirewallOff(ctx, cmd)
				},
			},
			{
				Name:  "list",
				Usage: t.Get("List all firewall rules"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.FirewallList(ctx, cmd)
				},
			},
			{
				Name:      "port",
				Usage:     t.Get("Allow or remove a port rule"),
				ArgsUsage: t.Get("<port or port range, e.g. 8888 or 8000-9000>"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "protocol",
						Usage:   t.Get("Protocol (tcp, udp, tcp/udp)"),
						Aliases: []string{"p"},
						Value:   "tcp/udp",
					},
					&cli.BoolFlag{
						Name:  "remove",
						Usage: t.Get("Remove the rule instead of adding it"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.FirewallPort(ctx, cmd)
				},
			},
		},
	}
}
