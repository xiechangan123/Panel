package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// StatusCommand 查询服务状态
func StatusCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "status",
		Usage: t.Get("Get AcePanel service status"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return do.MustInvoke[*service.CliService](i).Status(ctx, cmd)
		},
	}, nil
}

// RestartCommand 重启服务
func RestartCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "restart",
		Usage: t.Get("Restart AcePanel service"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return do.MustInvoke[*service.CliService](i).Restart(ctx, cmd)
		},
	}, nil
}

// StopCommand 停止服务
func StopCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "stop",
		Usage: t.Get("Stop AcePanel service"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return do.MustInvoke[*service.CliService](i).Stop(ctx, cmd)
		},
	}, nil
}

// StartCommand 启动服务
func StartCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "start",
		Usage: t.Get("Start AcePanel service"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return do.MustInvoke[*service.CliService](i).Start(ctx, cmd)
		},
	}, nil
}
