package job

import (
	"github.com/libtnb/cron"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/registry"
)

const Prefix = "jobs:"

// JobFn 向调度器注册一个任务。
type JobFn func(c *cron.Cron) error

// Package 装配定时任务层。
var Package = do.Package(
	do.LazyNamed(Prefix+"monitoring", MonitoringJob), do.LazyNamed(Prefix+"firewall_scan", FirewallScanJob),
	do.LazyNamed(Prefix+"cert_renew", CertRenewJob), do.LazyNamed(Prefix+"panel_task", PanelTaskJob),
	do.LazyNamed(Prefix+"website_stat", WebsiteStatJob), do.LazyNamed(Prefix+"website_expire", WebsiteExpireJob),
)

// Register 收集并注册全部任务贡献到调度器。
func Register(i do.Injector, c *cron.Cron) error {
	jobs, err := registry.Collect[JobFn](i, Prefix)
	if err != nil {
		return err
	}
	for _, reg := range jobs {
		if err := reg(c); err != nil {
			return err
		}
	}

	return nil
}
