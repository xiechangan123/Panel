package command

import (
	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

// Commands 组装全部 CLI 命令。
func Commands(t *gotext.Locale, cliService *service.CliService) []*cli.Command {
	return []*cli.Command{
		StatusCommand(t, cliService),
		RestartCommand(t, cliService),
		StopCommand(t, cliService),
		StartCommand(t, cliService),
		UpdateCommand(t, cliService),
		SyncCommand(t, cliService),
		FixCommand(t, cliService),
		InfoCommand(t, cliService),
		PortCommand(t, cliService),
		SyncTimeCommand(t, cliService),
		ClearTaskCommand(t, cliService),
		InitCommand(t, cliService),
		UserCommand(t, cliService),
		HttpsCommand(t, cliService),
		EntranceCommand(t, cliService),
		BindDomainCommand(t, cliService),
		BindIPCommand(t, cliService),
		BindUACommand(t, cliService),
		FirewallCommand(t, cliService),
		WebsiteCommand(t, cliService),
		CertCommand(t, cliService),
		DatabaseCommand(t, cliService),
		BackupCommand(t, cliService),
		RestoreCommand(t, cliService),
		CutoffCommand(t, cliService),
		CronCommand(t, cliService),
		AppCommand(t, cliService),
		SettingCommand(t, cliService),
		WebserverCommand(t, cliService),
	}
}

// 位置参数的用法文本沿用帮助输出的括号约定：尖括号必填、方括号可选，交互模式据此判断是否必填

// arg 必填的单值位置参数
func arg(name, usage string) cli.Argument {
	return &cli.StringArg{Name: name, UsageText: "<" + usage + ">"}
}

// optArg 可选的单值位置参数
func optArg(name, usage string) cli.Argument {
	return &cli.StringArg{Name: name, UsageText: "[" + usage + "]"}
}

// multiArg 至少一个的多值位置参数
func multiArg(name, usage string) cli.Argument {
	return &cli.StringArgs{Name: name, UsageText: "<" + usage + "> [" + usage + "...]", Max: -1}
}
