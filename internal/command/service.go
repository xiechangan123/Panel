package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// StatusCommand 查询服务状态
func StatusCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "status",
		Usage: t.Get("Get AcePanel service status"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cliService.Status(ctx, cmd)
		},
	}
}

// RestartCommand 重启服务
func RestartCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "restart",
		Usage: t.Get("Restart AcePanel service"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cliService.Restart(ctx, cmd)
		},
	}
}

// StopCommand 停止服务
func StopCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "stop",
		Usage: t.Get("Stop AcePanel service"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cliService.Stop(ctx, cmd)
		},
	}
}

// StartCommand 启动服务
func StartCommand(t *gotext.Locale, cliService *service.CliService) *cli.Command {

	return &cli.Command{
		Name:  "start",
		Usage: t.Get("Start AcePanel service"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cliService.Start(ctx, cmd)
		},
	}
}
