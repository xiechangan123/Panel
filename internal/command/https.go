package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// HttpsCommand 面板 HTTPS 管理命令组
func HttpsCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "https",
		Usage: t.Get("Operate AcePanel HTTPS"),
		Commands: []*cli.Command{
			{
				Name:  "on",
				Usage: t.Get("Enable HTTPS"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).HTTPSOn(ctx, cmd)
				},
			},
			{
				Name:  "off",
				Usage: t.Get("Disable HTTPS"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).HTTPSOff(ctx, cmd)
				},
			},
			{
				Name:  "generate",
				Usage: t.Get("Obtain a free certificate or generate a self-signed certificate"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).HTTPSGenerate(ctx, cmd)
				},
			},
		},
	}, nil
}
