package supervisor

import (
	"github.com/samber/lo"

	"github.com/acepanel/panel/v3/internal/apps/confval"
	"github.com/acepanel/panel/v3/pkg/os"
)

// mainConfPath supervisor 主配置，RHEL 系与 Debian 系的路径不同
func mainConfPath() string {
	return lo.Ternary(os.IsRHEL(), "/etc/supervisord.conf", "/etc/supervisor/supervisord.conf")
}

func confPath(name string) string {
	return lo.Ternary(os.IsRHEL(), "/etc/supervisord.d/", "/etc/supervisor/conf.d/") + name + ".conf"
}

type settingField struct {
	key   string
	value *string
}

func settingFields(s *ProcessSetting) []settingField {
	return []settingField{
		{"command", &s.Command},
		{"directory", &s.Directory},
		{"user", &s.User},
		{"numprocs", &s.NumProcs},
		{"priority", &s.Priority},
		{"autostart", &s.AutoStart},
		{"autorestart", &s.AutoRestart},
		{"startsecs", &s.StartSecs},
		{"startretries", &s.StartRetries},
		{"stopwaitsecs", &s.StopWaitSecs},
		{"stopasgroup", &s.StopAsGroup},
		{"killasgroup", &s.KillAsGroup},
		{"redirect_stderr", &s.RedirectStderr},
		{"stdout_logfile", &s.StdoutLogfile},
		{"stdout_logfile_maxbytes", &s.StdoutLogfileMaxBytes},
		{"stdout_logfile_backups", &s.StdoutLogfileBackups},
		{"environment", &s.Environment},
	}
}

func readSetting(config string, setting *ProcessSetting) {
	for _, f := range settingFields(setting) {
		*f.value = confval.Supervisor.Get(config, f.key)
	}
}

// writeSetting numprocs 大于 1 时 process_name 必须带上 process_num，否则 supervisor 会拒绝加载
func writeSetting(config string, setting *ProcessSetting) string {
	for _, f := range settingFields(setting) {
		config = confval.Supervisor.Set(config, f.key, *f.value)
	}

	return confval.Supervisor.Set(config, "process_name", processNameExpr(setting.NumProcs))
}

func processNameExpr(numProcs string) string {
	return lo.Ternary(numProcs == "" || numProcs == "1", `%(program_name)s`, `%(program_name)s_%(process_num)02d`)
}
