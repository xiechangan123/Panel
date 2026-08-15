package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// CertCommand 证书管理命令组
func CertCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "cert",
		Usage: t.Get("Certificate management"),
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: t.Get("List all certificates"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.CertList(ctx, cmd)
				},
			},
			{
				Name:  "renew",
				Usage: t.Get("Renew certificate"),
				Flags: []cli.Flag{
					&cli.UintFlag{
						Name:    "id",
						Usage:   t.Get("Certificate ID"),
						Aliases: []string{"i"},
					},
					&cli.BoolFlag{
						Name:  "all",
						Usage: t.Get("Renew all certificates with auto renewal enabled"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.CertRenew(ctx, cmd)
				},
			},
		},
	}
}
