package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// UserCommand 用户管理命令组
func UserCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "user",
		Usage: t.Get("Operate AcePanel users"),
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: t.Get("List all users"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.UserList(ctx, cmd)
				},
			},
			{
				Name:      "create",
				Usage:     t.Get("Create a new user (password is read from the ACEPANEL_PASSWORD environment variable or entered interactively if omitted)"),
				ArgsUsage: t.Get("<username> [password]"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "email",
						Usage:   t.Get("User email (generated from the username if not filled)"),
						Aliases: []string{"e"},
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.UserCreate(ctx, cmd)
				},
			},
			{
				Name:      "delete",
				Usage:     t.Get("Delete a user"),
				ArgsUsage: t.Get("<username>"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.UserDelete(ctx, cmd)
				},
			},
			{
				Name:      "username",
				Usage:     t.Get("Change a user's username"),
				ArgsUsage: t.Get("<old-username> <new-username>"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.UserName(ctx, cmd)
				},
			},
			{
				Name:      "password",
				Usage:     t.Get("Change a user's password (password is read from the ACEPANEL_PASSWORD environment variable or entered interactively if omitted)"),
				ArgsUsage: t.Get("<username> [password]"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.UserPassword(ctx, cmd)
				},
			},
			{
				Name:      "2fa",
				Usage:     t.Get("Toggle two-factor authentication for a user"),
				ArgsUsage: t.Get("<username>"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.UserTwoFA(ctx, cmd)
				},
			},
			{
				Name:      "passkey",
				Usage:     t.Get("Clear all passkeys for a user"),
				ArgsUsage: t.Get("<username>"),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return cliService.UserPasskey(ctx, cmd)
				},
			},
		},
	}
}
