package job

import (
	"log/slog"

	"github.com/google/wire"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/pkg/config"
	"github.com/acepanel/panel/pkg/websitestat"
)

var ProviderSet = wire.NewSet(NewJobs)

type Jobs struct {
	conf        *config.Config
	db          *gorm.DB
	log         *slog.Logger
	setting     biz.SettingRepo
	cert        biz.CertRepo
	certAccount biz.CertAccountRepo
	backup      biz.BackupRepo
	cache       biz.CacheRepo
	task        biz.TaskRepo
	scanRepo    biz.ScanEventRepo
	statRepo    biz.WebsiteStatRepo
	aggregator  *websitestat.Aggregator
}

func NewJobs(conf *config.Config, db *gorm.DB, log *slog.Logger, setting biz.SettingRepo, cert biz.CertRepo, certAccount biz.CertAccountRepo, backup biz.BackupRepo, cache biz.CacheRepo, task biz.TaskRepo, scanRepo biz.ScanEventRepo, statRepo biz.WebsiteStatRepo, aggregator *websitestat.Aggregator) *Jobs {
	return &Jobs{
		conf:        conf,
		db:          db,
		log:         log,
		setting:     setting,
		cert:        cert,
		certAccount: certAccount,
		backup:      backup,
		cache:       cache,
		task:        task,
		scanRepo:    scanRepo,
		statRepo:    statRepo,
		aggregator:  aggregator,
	}
}

func (r *Jobs) Register(c *cron.Cron) error {
	if _, err := c.AddJob("* * * * *", NewMonitoring(r.db, r.log, r.setting)); err != nil {
		return err
	}
	if _, err := c.AddJob("*/2 * * * *", NewFirewallScan(r.log, r.setting, r.scanRepo)); err != nil {
		return err
	}
	if _, err := c.AddJob("0 4 * * *", NewCertRenew(r.conf, r.db, r.log, r.setting, r.cert, r.certAccount)); err != nil {
		return err
	}
	if _, err := c.AddJob("0 2 * * *", NewPanelTask(r.conf, r.db, r.log, r.backup, r.cache, r.task, r.setting)); err != nil {
		return err
	}
	if _, err := c.AddJob("* * * * *", NewWebsiteStat(r.log, r.setting, r.statRepo, r.aggregator)); err != nil {
		return err
	}

	return nil
}
