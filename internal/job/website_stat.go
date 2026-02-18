package job

import (
	"log/slog"
	"strings"
	"time"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/pkg/websitestat"
)

// WebsiteStat 网站统计后台任务
type WebsiteStat struct {
	log        *slog.Logger
	setting    biz.SettingRepo
	statRepo   biz.WebsiteStatRepo
	aggregator *websitestat.Aggregator
	listener   *websitestat.Listener
	started    bool
}

// NewWebsiteStat 创建网站统计任务
func NewWebsiteStat(log *slog.Logger, setting biz.SettingRepo, statRepo biz.WebsiteStatRepo, aggregator *websitestat.Aggregator) *WebsiteStat {
	return &WebsiteStat{
		log:        log,
		setting:    setting,
		statRepo:   statRepo,
		aggregator: aggregator,
	}
}

func (r *WebsiteStat) Run() {
	if app.Status != app.StatusNormal {
		return
	}

	r.ensureListener()
	r.flush()
	r.flushErrors()
	r.flushDetails()
	r.cleanup()
}

// ensureListener 确保 listener goroutine 已启动
func (r *WebsiteStat) ensureListener() {
	if r.started {
		return
	}

	if v, err := r.setting.GetInt(biz.SettingKeyWebsiteStatErrBufMax); err == nil && v > 0 {
		r.aggregator.ErrBufMaxSize = v
	}
	if v, err := r.setting.GetInt(biz.SettingKeyWebsiteStatUVMaxKeys); err == nil && v > 0 {
		r.aggregator.UVMaxKeys = v
	}
	if v, err := r.setting.GetInt(biz.SettingKeyWebsiteStatIPMaxKeys); err == nil && v > 0 {
		r.aggregator.IPMaxKeys = v
	}
	if v, err := r.setting.GetInt(biz.SettingKeyWebsiteStatDetailMaxKeys); err == nil && v > 0 {
		r.aggregator.DetailMaxKeys = v
	}

	listener, err := websitestat.NewListener("/tmp/ace_stats.sock", r.log)
	if err != nil {
		r.log.Warn("fail to start website stat listener", slog.Any("err", err))
		return
	}

	r.listener = listener
	r.started = true

	go r.readLoop()
}

// readLoop 持续读取 syslog 消息并记录到聚合器
func (r *WebsiteStat) readLoop() {
	for {
		tag, data, err := r.listener.Read()
		if err != nil {
			r.log.Warn("failed to read from website stat listener", slog.Any("err", err))
			r.started = false
			return
		}

		entry, err := websitestat.ParseLogEntry(tag, data)
		if err != nil {
			continue
		}

		r.aggregator.Record(entry)
	}
}

// flush 将内存快照写入数据库（含每日汇总和小时数据）
func (r *WebsiteStat) flush() {
	date, snapshots := r.aggregator.Snapshot()
	if len(snapshots) == 0 {
		return
	}

	now := time.Now()
	var stats []*biz.WebsiteStat

	for site, snap := range snapshots {
		// 每日汇总行 (hour = -1)
		stats = append(stats, &biz.WebsiteStat{
			Site:      site,
			Date:      date,
			Hour:      -1,
			PV:        snap.PV,
			UV:        snap.UV,
			IP:        snap.IP,
			Bandwidth: snap.Bandwidth,
			Requests:  snap.Requests,
			Errors:    snap.Errors,
			Spiders:   snap.Spiders,
			UpdatedAt: now,
		})

		// 每小时行 (hour = 0-23)
		for h, hs := range snap.Hours {
			if hs == nil {
				continue
			}
			stats = append(stats, &biz.WebsiteStat{
				Site:      site,
				Date:      date,
				Hour:      h,
				PV:        hs.PV,
				UV:        hs.UV,
				IP:        hs.IP,
				Bandwidth: hs.Bandwidth,
				Requests:  hs.Requests,
				Errors:    hs.Errors,
				Spiders:   hs.Spiders,
				UpdatedAt: now,
			})
		}
	}

	if err := r.statRepo.Upsert(stats); err != nil {
		r.log.Warn("failed to upsert website stats", slog.Any("err", err))
	}
}

// flushErrors 将错误日志缓冲写入数据库
func (r *WebsiteStat) flushErrors() {
	entries := r.aggregator.DrainErrors()
	if len(entries) == 0 {
		return
	}

	now := time.Now()
	errors := make([]*biz.WebsiteErrorLog, 0, len(entries))
	for _, e := range entries {
		errors = append(errors, &biz.WebsiteErrorLog{
			Site:      e.Site,
			URI:       e.URI,
			Method:    e.Method,
			Status:    e.Status,
			IP:        e.IP,
			UA:        e.UA,
			Body:      e.Body,
			CreatedAt: now,
		})
	}

	if err := r.statRepo.InsertErrors(errors); err != nil {
		r.log.Warn("failed to insert website error logs", slog.Any("err", err))
	}
}

// flushDetails 将详细统计增量写入数据库（蜘蛛/客户端/IP/URI）
func (r *WebsiteStat) flushDetails() {
	date, details := r.aggregator.DrainDetailStats()
	if len(details) == 0 {
		return
	}

	now := time.Now()

	var spiders []*biz.WebsiteStatSpider
	var clients []*biz.WebsiteStatClient
	var ips []*biz.WebsiteStatIP
	var uris []*biz.WebsiteStatURI

	for site, snap := range details {
		for spider, requests := range snap.Spiders {
			spiders = append(spiders, &biz.WebsiteStatSpider{
				Site:      site,
				Date:      date,
				Spider:    spider,
				Requests:  requests,
				UpdatedAt: now,
			})
		}

		for key, cc := range snap.Clients {
			parts := strings.SplitN(key, "|", 2)
			if len(parts) != 2 {
				continue
			}
			clients = append(clients, &biz.WebsiteStatClient{
				Site:      site,
				Date:      date,
				Browser:   parts[0],
				OS:        parts[1],
				Requests:  cc.Requests,
				UpdatedAt: now,
			})
		}

		for ip, ic := range snap.IPs {
			ips = append(ips, &biz.WebsiteStatIP{
				Site:      site,
				Date:      date,
				IP:        ip,
				Requests:  ic.Requests,
				Bandwidth: ic.Bandwidth,
				UpdatedAt: now,
			})
		}

		for uri, uc := range snap.URIs {
			uris = append(uris, &biz.WebsiteStatURI{
				Site:      site,
				Date:      date,
				URI:       uri,
				Requests:  uc.Requests,
				Bandwidth: uc.Bandwidth,
				Errors:    uc.Errors,
				UpdatedAt: now,
			})
		}
	}

	if err := r.statRepo.UpsertSpiders(spiders); err != nil {
		r.log.Warn("failed to upsert spider stats", slog.Any("err", err))
	}
	if err := r.statRepo.UpsertClients(clients); err != nil {
		r.log.Warn("failed to upsert client stats", slog.Any("err", err))
	}
	if err := r.statRepo.UpsertIPs(ips); err != nil {
		r.log.Warn("failed to upsert ip stats", slog.Any("err", err))
	}
	if err := r.statRepo.UpsertURIs(uris); err != nil {
		r.log.Warn("failed to upsert uri stats", slog.Any("err", err))
	}
}

// cleanup 清理过期数据
func (r *WebsiteStat) cleanup() {
	days, err := r.setting.GetInt(biz.SettingKeyWebsiteStatDays, 30)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -days).Format(time.DateOnly)
	if err = r.statRepo.ClearBefore(cutoff); err != nil {
		r.log.Warn("failed to clear expired website stats", slog.Any("err", err))
	}

	errCutoff := time.Now().AddDate(0, 0, -days)
	if err = r.statRepo.ClearErrorsBefore(errCutoff); err != nil {
		r.log.Warn("failed to clear expired website error logs", slog.Any("err", err))
	}

	// 清理详细统计表
	if err = r.statRepo.ClearSpidersBefore(cutoff); err != nil {
		r.log.Warn("failed to clear expired spider stats", slog.Any("err", err))
	}
	if err = r.statRepo.ClearClientsBefore(cutoff); err != nil {
		r.log.Warn("failed to clear expired client stats", slog.Any("err", err))
	}
	if err = r.statRepo.ClearIPsBefore(cutoff); err != nil {
		r.log.Warn("failed to clear expired ip stats", slog.Any("err", err))
	}
	if err = r.statRepo.ClearURIsBefore(cutoff); err != nil {
		r.log.Warn("failed to clear expired uri stats", slog.Any("err", err))
	}
}
