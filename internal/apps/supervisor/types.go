package supervisor

type Process struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Pid    string `json:"pid"`
	Uptime string `json:"uptime"`
}

// ProcessSetting 进程的可视化参数，留空表示注释掉该项、走 supervisor 默认值
type ProcessSetting struct {
	Command               string `form:"command" json:"command" validate:"required"`
	Directory             string `form:"directory" json:"directory"`
	User                  string `form:"user" json:"user"`
	NumProcs              string `form:"numprocs" json:"numprocs"`
	Priority              string `form:"priority" json:"priority"`
	AutoStart             string `form:"autostart" json:"autostart"`
	AutoRestart           string `form:"autorestart" json:"autorestart"`
	StartSecs             string `form:"startsecs" json:"startsecs"`
	StartRetries          string `form:"startretries" json:"startretries"`
	StopWaitSecs          string `form:"stopwaitsecs" json:"stopwaitsecs"`
	StopAsGroup           string `form:"stopasgroup" json:"stopasgroup"`
	KillAsGroup           string `form:"killasgroup" json:"killasgroup"`
	RedirectStderr        string `form:"redirect_stderr" json:"redirect_stderr"`
	StdoutLogfile         string `form:"stdout_logfile" json:"stdout_logfile"`
	StdoutLogfileMaxBytes string `form:"stdout_logfile_maxbytes" json:"stdout_logfile_maxbytes"`
	StdoutLogfileBackups  string `form:"stdout_logfile_backups" json:"stdout_logfile_backups"`
	Environment           string `form:"environment" json:"environment"`
}
