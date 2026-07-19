package job

import (
	"github.com/libtnb/cron"
	"github.com/samber/do/v2"
)

const Prefix = "jobs:"

// Job 声明一个定时任务
type Job struct {
	Spec string   // cron 表达式
	Task cron.Job // 任务体
}

var Package = do.Package(
	do.LazyNamed(Prefix+"monitoring", NewMonitoring),
	do.LazyNamed(Prefix+"firewall_scan", NewFirewallScan),
	do.LazyNamed(Prefix+"cert_renew", NewCertRenew),
	do.LazyNamed(Prefix+"panel_task", NewPanelTask),
	do.LazyNamed(Prefix+"website_stat", NewWebsiteStat),
	do.LazyNamed(Prefix+"website_expire", NewWebsiteExpire),
	do.LazyNamed(Prefix+"tamper", NewTamper),
)
