package command

import (
	"github.com/google/wire"
	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v3"

	"github.com/acepanel/panel/v3/internal/service"
)

var ProviderSet = wire.NewSet(Commands)

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
		WebsiteCommand(t, cliService),
		DatabaseCommand(t, cliService),
		BackupCommand(t, cliService),
		RestoreCommand(t, cliService),
		CutoffCommand(t, cliService),
		CronCommand(t, cliService),
		AppCommand(t, cliService),
		SettingCommand(t, cliService),
	}
}
