package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// UserCommand 用户管理命令组
func UserCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "user",
		Usage: t.Get("Operate AcePanel users"),
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: t.Get("List all users"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).UserList(ctx, cmd)
				},
			},
			{
				Name:  "username",
				Usage: t.Get("Change a user's username"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).UserName(ctx, cmd)
				},
			},
			{
				Name:  "password",
				Usage: t.Get("Change a user's password"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).UserPassword(ctx, cmd)
				},
			},
			{
				Name:  "2fa",
				Usage: t.Get("Toggle two-factor authentication for a user"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).UserTwoFA(ctx, cmd)
				},
			},
			{
				Name:  "passkey",
				Usage: t.Get("Clear all passkeys for a user"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return do.MustInvoke[*service.CliService](i).UserPasskey(ctx, cmd)
				},
			},
		},
	}, nil
}
