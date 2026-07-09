package command

import (
	"context"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// UpdateCommand 更新面板
func UpdateCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "update",
		Usage: t.Get("Update AcePanel to the latest version"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return do.MustInvoke[*service.CliService](i).Update(ctx, cmd)
		},
	}, nil
}

// SyncCommand 同步云端缓存数据
func SyncCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "sync",
		Usage: t.Get("Sync AcePanel cached data with cloud"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return do.MustInvoke[*service.CliService](i).Sync(ctx, cmd)
		},
	}, nil
}

// FixCommand 修复升级问题
func FixCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "fix",
		Usage: t.Get("Fix AcePanel upgrade issues"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return do.MustInvoke[*service.CliService](i).Fix(ctx, cmd)
		},
	}, nil
}

// InfoCommand 输出面板基础信息
func InfoCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "info",
		Usage: t.Get("Output AcePanel basic information"),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   t.Get("Force reset password"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return do.MustInvoke[*service.CliService](i).Info(ctx, cmd)
		},
	}, nil
}

// PortCommand 修改监听端口
func PortCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "port",
		Usage: t.Get("Change the AcePanel listening port"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return do.MustInvoke[*service.CliService](i).Port(ctx, cmd)
		},
	}, nil
}

// SyncTimeCommand 通过 NTP 同步系统时间
func SyncTimeCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:  "sync-time",
		Usage: t.Get("Sync server time with NTP"),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return do.MustInvoke[*service.CliService](i).SyncTime(ctx, cmd)
		},
	}, nil
}

// ClearTaskCommand 清理卡住的任务队列
func ClearTaskCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:   "clear-task",
		Usage:  t.Get("Clear all tasks in the task queue if they are stuck (use only under guidance)"),
		Hidden: true,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return do.MustInvoke[*service.CliService](i).ClearTask(ctx, cmd)
		},
	}, nil
}

// InitCommand 初始化面板
func InitCommand(i do.Injector) (*cli.Command, error) {
	t := do.MustInvoke[*gotext.Locale](i)
	return &cli.Command{
		Name:   "init",
		Usage:  t.Get("Initialize AcePanel (use only under guidance)"),
		Hidden: true,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return do.MustInvoke[*service.CliService](i).Init(ctx, cmd)
		},
	}, nil
}
