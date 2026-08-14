package biz

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	"github.com/spf13/cast"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/apploader"
	"github.com/acepanel/panel/v3/pkg/shell"
	"github.com/acepanel/panel/v3/pkg/sshlog"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/tools"
	"github.com/acepanel/panel/v3/pkg/types"
)

// 告警指标类型
const (
	AlertTypeCPU           = "cpu"            // CPU 使用率 %
	AlertTypeMemory        = "memory"         // 内存使用率 %
	AlertTypeSwap          = "swap"           // Swap 使用率 %
	AlertTypeLoad1         = "load1"          // 1 分钟平均负载
	AlertTypeLoad5         = "load5"          // 5 分钟平均负载
	AlertTypeLoad15        = "load15"         // 15 分钟平均负载
	AlertTypeDisk          = "disk"           // 磁盘使用率 %，目标为挂载点
	AlertTypeDiskInode     = "disk_inode"     // 磁盘 inode 使用率 %，目标为挂载点
	AlertTypeDiskRead      = "disk_read"      // 磁盘读取速率 MB/s，目标为设备名
	AlertTypeDiskWrite     = "disk_write"     // 磁盘写入速率 MB/s，目标为设备名
	AlertTypeNetIn         = "net_in"         // 网卡下行速率 MB/s，目标为网卡名
	AlertTypeNetOut        = "net_out"        // 网卡上行速率 MB/s，目标为网卡名
	AlertTypeWebsite5xx    = "website_5xx"    // 网站本小时 5xx 次数，目标为网站名
	AlertTypeWebsiteError  = "website_error"  // 网站本小时错误率 %，目标为网站名
	AlertTypeService       = "service"        // 服务未运行，目标为服务名
	AlertTypeProject       = "project"        // 项目未运行，目标为项目名
	AlertTypeContainer     = "container"      // 容器未运行，目标为容器名
	AlertTypeApp           = "app"            // 应用未运行，目标为应用标识
	AlertTypeDatabase      = "database"       // 数据库服务器不可达，目标为服务器名
	AlertTypeCertExpire    = "cert_expire"    // 证书剩余天数，目标为域名
	AlertTypeWebsiteExpire = "website_expire" // 网站剩余天数，目标为网站名
)

// statusAlertTypes 状态类指标
var statusAlertTypes = []string{AlertTypeService, AlertTypeProject, AlertTypeContainer, AlertTypeApp, AlertTypeDatabase}

const (
	// alertRetryDelay 通知发送失败后的重试间隔
	alertRetryDelay = 5 * time.Minute
	// sshFailThreshold 单次检查窗口内触发爆破告警的失败次数
	sshFailThreshold uint = 5
	// sshFailSilence 同一来源两次爆破告警的最小间隔
	sshFailSilence = 30 * time.Minute
)

// 告警比较运算符
const (
	AlertOperatorGT  = "gt"
	AlertOperatorGTE = "gte"
	AlertOperatorLT  = "lt"
	AlertOperatorLTE = "lte"
)

// AlertRule 告警规则
type AlertRule struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null;default:''" json:"name"`
	Type      string    `gorm:"not null;default:''" json:"type"`
	Target    string    `gorm:"not null;default:''" json:"target"`     // 挂载点/网卡/磁盘/服务名/域名，空表示全部
	Operator  string    `gorm:"not null;default:'gt'" json:"operator"` // gt/gte/lt/lte
	Threshold float64   `gorm:"not null;default:0" json:"threshold"`
	Duration  uint      `gorm:"not null;default:1" json:"duration"` // 连续满足次数
	Silence   uint      `gorm:"not null;default:30" json:"silence"` // 静默期（分钟）
	Channels  []uint    `gorm:"serializer:json" json:"channels"`    // 通知渠道 ID
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Alert 告警记录
type Alert struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RuleID    uint      `gorm:"not null;default:0;index" json:"rule_id"`
	RuleName  string    `gorm:"not null;default:''" json:"rule_name"`
	Type      string    `gorm:"not null;default:''" json:"type"`
	Target    string    `gorm:"not null;default:''" json:"target"`
	Value     float64   `gorm:"not null;default:0" json:"value"`
	Message   string    `gorm:"not null;default:''" json:"message"`
	Notified  bool      `gorm:"not null;default:false" json:"notified"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// AlertMetric 单个目标的取值
type AlertMetric struct {
	Target string
	Value  float64
}

type AlertRepo interface {
	ListRules(page, limit uint) ([]*AlertRule, int64, error)
	AllRules() ([]*AlertRule, error)
	GetRule(id uint) (*AlertRule, error)
	CreateRule(rule *AlertRule) error
	UpdateRule(rule *AlertRule) error
	DeleteRule(id uint) error
	AddAlert(alert *Alert) error
	ListAlerts(page, limit uint) ([]*Alert, int64, error)
	ClearAlerts() error
	ClearAlertsBefore(t time.Time) error
	CertExpiry() ([]*AlertMetric, error)
	WebsiteExpiry() ([]*AlertMetric, error)
	ProjectNames() ([]string, error)
	DatabaseServers() ([]*DatabaseServer, error)
	WebsiteHourStats() ([]*WebsiteHourStat, error)
}

// WebsiteHourStat 网站当前小时的请求统计
type WebsiteHourStat struct {
	Site      string
	Requests  uint64
	Errors    uint64
	Status5xx uint64
}

// ioSnapshot 网卡/磁盘累计计数快照及据此算出的速率（字节/秒）
type ioSnapshot struct {
	in      uint64
	out     uint64
	inRate  uint64
	outRate uint64
	at      time.Time
}

type AlertUsecase struct {
	repo      AlertRepo
	notify    *NotifyUsecase
	setting   SettingRepo
	container ContainerRepo
	app       AppRepo
	database  DatabaseServerRepo
	loader    *apploader.Loader
	log       *slog.Logger
	t         *gotext.Locale

	mu         sync.Mutex
	hits       map[string]uint       // 连续命中次数
	fired      map[string]time.Time  // 上次通知时间
	netSnaps   map[string]ioSnapshot // 网卡累计流量快照
	diskSnaps  map[string]ioSnapshot // 磁盘累计 IO 快照
	healthKeys map[string]struct{}   // 已通知的健康问题
	sshFired   map[string]time.Time  // SSH 爆破上次通知时间
	sshAt      time.Time             // SSH 日志上次检查时间
	dbProbe    map[string]bool       // 本轮数据库连通性探测结果
	cleanedAt  time.Time
}

func NewAlertUsecase(notifyUsecase *NotifyUsecase, loader *apploader.Loader, t *gotext.Locale, log *slog.Logger, alertRepo AlertRepo, appRepo AppRepo, containerRepo ContainerRepo, databaseServerRepo DatabaseServerRepo, settingRepo SettingRepo) (*AlertUsecase, error) {
	return &AlertUsecase{
		repo:       alertRepo,
		notify:     notifyUsecase,
		setting:    settingRepo,
		container:  containerRepo,
		app:        appRepo,
		database:   databaseServerRepo,
		loader:     loader,
		log:        log,
		t:          t,
		hits:       make(map[string]uint),
		fired:      make(map[string]time.Time),
		netSnaps:   make(map[string]ioSnapshot),
		diskSnaps:  make(map[string]ioSnapshot),
		healthKeys: make(map[string]struct{}),
		sshFired:   make(map[string]time.Time),
	}, nil
}

func (uc *AlertUsecase) ListRules(page, limit uint) ([]*AlertRule, int64, error) {
	return uc.repo.ListRules(page, limit)
}

func (uc *AlertUsecase) GetRule(id uint) (*AlertRule, error) {
	return uc.repo.GetRule(id)
}

func (uc *AlertUsecase) CreateRule(ctx context.Context, req *request.AlertRuleCreate) (*AlertRule, error) {
	rule := &AlertRule{
		Name:      req.Name,
		Type:      req.Type,
		Target:    req.Target,
		Operator:  req.Operator,
		Threshold: req.Threshold,
		Duration:  req.Duration,
		Silence:   req.Silence,
		Channels:  req.Channels,
		Enabled:   req.Enabled,
	}
	normalizeRule(rule)

	if err := uc.repo.CreateRule(rule); err != nil {
		return nil, err
	}

	uc.log.Info("alert rule created", slog.String("type", OperationTypeMonitor), slog.Uint64("operator_id", operatorID(ctx)), slog.String("name", req.Name))

	return rule, nil
}

func (uc *AlertUsecase) UpdateRule(ctx context.Context, req *request.AlertRuleUpdate) error {
	rule, err := uc.repo.GetRule(req.ID)
	if err != nil {
		return err
	}

	rule.Name = req.Name
	rule.Type = req.Type
	rule.Target = req.Target
	rule.Operator = req.Operator
	rule.Threshold = req.Threshold
	rule.Duration = req.Duration
	rule.Silence = req.Silence
	rule.Channels = req.Channels
	rule.Enabled = req.Enabled
	normalizeRule(rule)

	if err = uc.repo.UpdateRule(rule); err != nil {
		return err
	}

	// 规则变更后重置命中状态，避免沿用旧阈值的计数
	uc.mu.Lock()
	uc.clearState(rule.ID)
	uc.mu.Unlock()

	uc.log.Info("alert rule updated", slog.String("type", OperationTypeMonitor), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(req.ID)), slog.String("name", req.Name))

	return nil
}

func (uc *AlertUsecase) DeleteRule(ctx context.Context, id uint) error {
	rule, err := uc.repo.GetRule(id)
	if err != nil {
		return err
	}
	if err = uc.repo.DeleteRule(id); err != nil {
		return err
	}

	uc.mu.Lock()
	uc.clearState(id)
	uc.mu.Unlock()

	uc.log.Info("alert rule deleted", slog.String("type", OperationTypeMonitor), slog.Uint64("operator_id", operatorID(ctx)), slog.Uint64("id", uint64(id)), slog.String("name", rule.Name))

	return nil
}

func (uc *AlertUsecase) ListAlerts(page, limit uint) ([]*Alert, int64, error) {
	return uc.repo.ListAlerts(page, limit)
}

func (uc *AlertUsecase) ClearAlerts() error {
	return uc.repo.ClearAlerts()
}

// Evaluate 评估全部启用的规则，命中则记录并通知
func (uc *AlertUsecase) Evaluate(ctx context.Context) error {
	rules, err := uc.repo.AllRules()
	if err != nil {
		return err
	}

	// 探测结果按轮缓存，进入新一轮先失效
	uc.mu.Lock()
	uc.dbProbe = nil
	uc.mu.Unlock()

	uc.cleanup()
	uc.checkHealth(ctx)
	uc.checkSSH(ctx)

	var enabled []*AlertRule
	for _, rule := range rules {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}
	if len(enabled) == 0 {
		uc.mu.Lock()
		clear(uc.hits)
		clear(uc.fired)
		uc.mu.Unlock()
		return nil
	}

	info := tools.CurrentInfo(nil, nil)
	now := time.Now()
	uc.updateSnapshots(info, now)

	alive := make(map[string]struct{})
	for _, rule := range enabled {
		metrics, err := uc.collect(ctx, rule, info)
		if err != nil {
			uc.log.Warn("failed to collect alert metric", slog.String("rule", rule.Name), slog.Any("err", err))
			continue
		}

		for _, metric := range metrics {
			key := stateKey(rule.ID, metric.Target)
			alive[key] = struct{}{}
			uc.evaluateMetric(ctx, rule, metric, key, now)
		}
	}

	// 清理已消失的目标状态
	uc.mu.Lock()
	for key := range uc.hits {
		if _, ok := alive[key]; !ok {
			delete(uc.hits, key)
		}
	}
	uc.mu.Unlock()

	return nil
}

func (uc *AlertUsecase) evaluateMetric(ctx context.Context, rule *AlertRule, metric *AlertMetric, key string, now time.Time) {
	uc.mu.Lock()
	if !matchThreshold(rule, metric.Value) {
		delete(uc.hits, key)
		uc.mu.Unlock()
		return
	}

	uc.hits[key]++
	hits := uc.hits[key]
	duration := max(rule.Duration, 1)
	if hits < duration {
		uc.mu.Unlock()
		return
	}

	// 静默期内不重复告警
	if last, ok := uc.fired[key]; ok && now.Sub(last) < time.Duration(rule.Silence)*time.Minute {
		uc.mu.Unlock()
		return
	}
	uc.fired[key] = now
	uc.mu.Unlock()

	alert := &Alert{
		RuleID:   rule.ID,
		RuleName: rule.Name,
		Type:     rule.Type,
		Target:   metric.Target,
		Value:    metric.Value,
		Message:  uc.buildMessage(rule, metric),
	}

	sent, err := uc.notify.Send(ctx, rule.Channels, uc.t.Get("[AcePanel] Alert: %s", rule.Name), NotifyBody(alert.Message, [][2]string{
		{uc.t.Get("Rule"), rule.Name},
		{uc.t.Get("Metric"), uc.metricLabel(rule.Type, metric.Target)},
		{uc.t.Get("Current Value"), uc.formatValue(rule.Type, metric.Value)},
		{uc.t.Get("Threshold"), uc.formatValue(rule.Type, rule.Threshold)},
		{uc.t.Get("Time"), now.Format(time.DateTime)},
	}))
	if err != nil {
		uc.log.Warn("failed to send alert notification", slog.String("rule", rule.Name), slog.Any("err", err))
	}
	alert.Notified = sent > 0

	// 一条都没送达时缩短静默期，让下一轮重试，避免临时故障吞掉整个静默窗口
	if len(rule.Channels) > 0 && sent == 0 {
		uc.mu.Lock()
		uc.fired[key] = now.Add(alertRetryDelay - time.Duration(rule.Silence)*time.Minute)
		uc.mu.Unlock()
	}

	if err := uc.repo.AddAlert(alert); err != nil {
		uc.log.Warn("failed to save alert record", slog.String("rule", rule.Name), slog.Any("err", err))
	}
}

// collect 采集规则对应的目标取值
func (uc *AlertUsecase) collect(ctx context.Context, rule *AlertRule, info types.CurrentInfo) ([]*AlertMetric, error) {
	switch rule.Type {
	case AlertTypeCPU:
		return []*AlertMetric{{Value: info.Percent}}, nil

	case AlertTypeMemory:
		if info.Mem == nil {
			return nil, nil
		}
		return []*AlertMetric{{Value: info.Mem.UsedPercent}}, nil

	case AlertTypeSwap:
		if info.Swap == nil || info.Swap.Total == 0 {
			return nil, nil
		}
		return []*AlertMetric{{Value: info.Swap.UsedPercent}}, nil

	case AlertTypeLoad1, AlertTypeLoad5, AlertTypeLoad15:
		if info.Load == nil {
			return nil, nil
		}
		switch rule.Type {
		case AlertTypeLoad1:
			return []*AlertMetric{{Value: info.Load.Load1}}, nil
		case AlertTypeLoad5:
			return []*AlertMetric{{Value: info.Load.Load5}}, nil
		default:
			return []*AlertMetric{{Value: info.Load.Load15}}, nil
		}

	case AlertTypeDisk, AlertTypeDiskInode:
		metrics := make([]*AlertMetric, 0)
		for _, usage := range info.DiskUsage {
			if rule.Target != "" && rule.Target != usage.Path {
				continue
			}
			value := usage.UsedPercent
			if rule.Type == AlertTypeDiskInode {
				value = usage.InodesUsedPercent
			}
			metrics = append(metrics, &AlertMetric{Target: usage.Path, Value: value})
		}
		return metrics, nil

	case AlertTypeNetIn, AlertTypeNetOut, AlertTypeDiskRead, AlertTypeDiskWrite:
		return uc.rateMetrics(rule), nil

	case AlertTypeWebsite5xx, AlertTypeWebsiteError:
		stats, err := uc.repo.WebsiteHourStats()
		if err != nil {
			return nil, err
		}
		metrics := make([]*AlertMetric, 0, len(stats))
		for _, item := range stats {
			if rule.Target != "" && rule.Target != item.Site {
				continue
			}
			if rule.Type == AlertTypeWebsite5xx {
				metrics = append(metrics, &AlertMetric{Target: item.Site, Value: float64(item.Status5xx)})
				continue
			}
			// 无请求时错误率无意义
			if item.Requests == 0 {
				continue
			}
			metrics = append(metrics, &AlertMetric{Target: item.Site, Value: float64(item.Errors) / float64(item.Requests) * 100})
		}
		return metrics, nil

	case AlertTypeService:
		if rule.Target == "" {
			return nil, nil
		}
		return []*AlertMetric{{Target: rule.Target, Value: notRunning(systemctl.Status(rule.Target))}}, nil

	case AlertTypeProject:
		names, err := uc.repo.ProjectNames()
		if err != nil {
			return nil, err
		}
		// 项目即 systemd 单元，单元名与项目名一致
		return lop.Map(filterNames(rule.Target, names), func(name string, _ int) *AlertMetric {
			return &AlertMetric{Target: name, Value: notRunning(systemctl.Status(name))}
		}), nil

	case AlertTypeContainer:
		containers, err := uc.container.ListAll(containerSock(uc.setting))
		if err != nil {
			return nil, err
		}
		metrics := make([]*AlertMetric, 0, len(containers))
		for _, item := range containers {
			if rule.Target != "" && rule.Target != item.Name {
				continue
			}
			value := float64(0)
			if item.State != "running" {
				value = 1
			}
			metrics = append(metrics, &AlertMetric{Target: item.Name, Value: value})
		}
		return metrics, nil

	case AlertTypeApp:
		installed, err := uc.app.Installed()
		if err != nil {
			return nil, err
		}
		targets := lo.Filter(installed, func(item *App, _ int) bool {
			if rule.Target != "" && rule.Target != item.Slug {
				return false
			}
			_, ok := uc.loader.Get(item.Slug)
			return ok
		})
		// 状态查询逐个访问 systemd，并发采集
		return lo.Compact(lop.Map(targets, func(item *App, _ int) *AlertMetric {
			a, _ := uc.loader.Get(item.Slug)
			status := a.Status()
			// 无 systemd 服务的应用没有运行状态
			if status == types.AppStatusNA {
				return nil
			}
			value := float64(0)
			if status != types.AppStatusRunning {
				value = 1
			}
			return &AlertMetric{Target: item.Slug, Value: value}
		})), nil

	case AlertTypeDatabase:
		servers, err := uc.repo.DatabaseServers()
		if err != nil {
			return nil, err
		}
		reachable := uc.probeDatabases(ctx, servers)

		metrics := make([]*AlertMetric, 0, len(servers))
		for _, item := range servers {
			if rule.Target != "" && rule.Target != item.Name {
				continue
			}
			value := float64(0)
			if !reachable[item.Name] {
				value = 1
			}
			metrics = append(metrics, &AlertMetric{Target: item.Name, Value: value})
		}
		return metrics, nil

	case AlertTypeCertExpire:
		metrics, err := uc.repo.CertExpiry()
		if err != nil {
			return nil, err
		}
		return filterTarget(rule.Target, metrics), nil

	case AlertTypeWebsiteExpire:
		metrics, err := uc.repo.WebsiteExpiry()
		if err != nil {
			return nil, err
		}
		return filterTarget(rule.Target, metrics), nil
	}

	return nil, fmt.Errorf("unsupported alert type: %s", rule.Type)
}

// notRunning 将运行状态转为告警取值，未运行记为 1，配合 normalizeRule 的 >=1 判定
func notRunning(running bool, _ error) float64 {
	if running {
		return 0
	}

	return 1
}

// filterNames 按目标名筛选，目标为空表示全部
func filterNames(target string, names []string) []string {
	if target == "" {
		return names
	}
	if slices.Contains(names, target) {
		return []string{target}
	}

	return nil
}

// filterTarget 按目标名筛选，目标为空表示全部
// 证书目标是逗号分隔的多域名，允许用其中任一域名命中
func filterTarget(target string, metrics []*AlertMetric) []*AlertMetric {
	if target == "" {
		return metrics
	}

	filtered := make([]*AlertMetric, 0)
	for _, metric := range metrics {
		if metric.Target == target || slices.Contains(strings.Split(metric.Target, ","), target) {
			filtered = append(filtered, metric)
		}
	}

	return filtered
}

// probeDatabases 并发探测数据库连通性，结果在本轮评估内复用，避免多条规则重复整批探测
// 单台耗时上限由 pkg/db 各驱动的连接超时保证（5~10 秒），并发后整批不会拖长分钟级评估
func (uc *AlertUsecase) probeDatabases(ctx context.Context, servers []*DatabaseServer) map[string]bool {
	uc.mu.Lock()
	cached := uc.dbProbe
	uc.mu.Unlock()
	if cached != nil {
		return cached
	}

	probe := lo.FromEntries(lop.Map(servers, func(item *DatabaseServer, _ int) lo.Entry[string, bool] {
		return lo.Entry[string, bool]{Key: item.Name, Value: uc.database.CheckServer(ctx, item)}
	}))

	uc.mu.Lock()
	uc.dbProbe = probe
	uc.mu.Unlock()

	return probe
}

// rateMetrics 从快照读取速率指标（MB/s）
func (uc *AlertUsecase) rateMetrics(rule *AlertRule) []*AlertMetric {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	snaps := uc.netSnaps
	if rule.Type == AlertTypeDiskRead || rule.Type == AlertTypeDiskWrite {
		snaps = uc.diskSnaps
	}

	metrics := make([]*AlertMetric, 0, len(snaps))
	for name, snap := range snaps {
		if rule.Target != "" && rule.Target != name {
			continue
		}
		value := snap.inRate
		if rule.Type == AlertTypeNetOut || rule.Type == AlertTypeDiskWrite {
			value = snap.outRate
		}
		metrics = append(metrics, &AlertMetric{Target: name, Value: float64(value) / 1024 / 1024})
	}

	return metrics
}

// updateSnapshots 用两次采集的累计值计算速率，结果暂存于快照
func (uc *AlertUsecase) updateSnapshots(info types.CurrentInfo, now time.Time) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	nets := make(map[string]ioSnapshot, len(info.Net))
	for _, item := range info.Net {
		if item.Name == "lo" {
			continue
		}
		nets[item.Name] = rateOf(uc.netSnaps[item.Name], item.BytesRecv, item.BytesSent, now)
	}
	uc.netSnaps = nets

	disks := make(map[string]ioSnapshot, len(info.DiskIO))
	for _, item := range info.DiskIO {
		disks[item.Name] = rateOf(uc.diskSnaps[item.Name], item.ReadBytes, item.WriteBytes, now)
	}
	uc.diskSnaps = disks
}

// checkHealth 上报新出现的面板健康问题，问题恢复后重新出现会再次通知
// 同步发送并只在送达后记入去重集合，否则一次发送失败就会让持续存在的问题再也不告警
func (uc *AlertUsecase) checkHealth(ctx context.Context) {
	issues := app.Health.Snapshot()

	uc.mu.Lock()
	fresh := make([]app.HealthIssue, 0, len(issues))
	notified := make(map[string]struct{}, len(issues))
	for _, issue := range issues {
		if _, ok := uc.healthKeys[issue.Key]; ok {
			notified[issue.Key] = struct{}{}
			continue
		}
		fresh = append(fresh, issue)
	}
	// 已消失的问题在此被丢弃，恢复后再次出现会重新通知
	uc.healthKeys = notified
	uc.mu.Unlock()

	for _, issue := range fresh {
		if err := uc.notify.SendEventSync(ctx, NotifyEventHealth, uc.t.Get("[AcePanel] Panel Health Issue"), NotifyBody(uc.t.Get("panel reported a health issue"), [][2]string{
			{uc.t.Get("Item"), issue.Key},
			{uc.t.Get("Level"), issue.Level},
			{uc.t.Get("Detail"), issue.Message},
			{uc.t.Get("Time"), issue.Since.Format(time.DateTime)},
		})); err != nil {
			uc.log.Warn("failed to send health notification", slog.String("item", issue.Key), slog.Any("err", err))
			continue
		}

		uc.mu.Lock()
		uc.healthKeys[issue.Key] = struct{}{}
		uc.mu.Unlock()
	}
}

// checkSSH 增量检查 sshd 日志，上报登录成功与爆破尝试
func (uc *AlertUsecase) checkSSH(ctx context.Context) {
	now := time.Now()

	uc.mu.Lock()
	since := uc.sshAt
	uc.sshAt = now
	uc.mu.Unlock()

	// 首次仅记录时间，避免面板启动时把历史日志全推一遍
	if since.IsZero() {
		return
	}

	raw, err := shell.ExecfWithContext(ctx, `journalctl -u sshd -u ssh --no-pager -o json --since "@%d" 2>/dev/null`, since.Unix())
	if err != nil {
		// 读取失败则回退检查点，下一轮重扫该窗口，避免丢掉这段时间的登录记录
		uc.mu.Lock()
		uc.sshAt = since
		uc.mu.Unlock()
		return
	}
	if raw == "" {
		return
	}

	failures := make(map[string]uint)
	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry struct {
			Message string `json:"MESSAGE"`
		}
		if json.Unmarshal(scanner.Bytes(), &entry) != nil {
			continue
		}

		record := sshlog.ParseMessage(entry.Message)
		if record == nil {
			continue
		}

		switch record.Status {
		case sshlog.StatusAccepted:
			uc.notify.SendEvent(NotifyEventSSHLogin, uc.t.Get("[AcePanel] SSH Login"), NotifyBody(uc.t.Get("SSH login succeeded"), [][2]string{
				{uc.t.Get("Username"), record.User},
				{uc.t.Get("IP"), record.IP},
				{uc.t.Get("Method"), record.Method},
				{uc.t.Get("Time"), now.Format(time.DateTime)},
			}))
		case sshlog.StatusFailed, sshlog.StatusInvalidUser:
			failures[record.IP]++
		}
	}

	for ip, count := range failures {
		if count < sshFailThreshold || !uc.sshShouldFire(ip, now) {
			continue
		}
		uc.notify.SendEvent(NotifyEventSSHBruteforce, uc.t.Get("[AcePanel] SSH Brute-force Attempts"), NotifyBody(uc.t.Get("too many failed SSH login attempts"), [][2]string{
			{uc.t.Get("IP"), ip},
			{uc.t.Get("Failed Attempts"), cast.ToString(count)},
			{uc.t.Get("Time"), now.Format(time.DateTime)},
		}))
	}
}

// sshShouldFire 判断某来源是否已过静默期，并顺带清理过期记录
func (uc *AlertUsecase) sshShouldFire(ip string, now time.Time) bool {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	for key, at := range uc.sshFired {
		if now.Sub(at) > sshFailSilence {
			delete(uc.sshFired, key)
		}
	}

	if at, ok := uc.sshFired[ip]; ok && now.Sub(at) < sshFailSilence {
		return false
	}
	uc.sshFired[ip] = now

	return true
}

// cleanup 按保留天数清理历史告警记录
func (uc *AlertUsecase) cleanup() {
	uc.mu.Lock()
	if time.Since(uc.cleanedAt) < 6*time.Hour {
		uc.mu.Unlock()
		return
	}
	uc.cleanedAt = time.Now()
	uc.mu.Unlock()

	days, err := uc.setting.GetInt(SettingKeyAlertLogDays, 30)
	if err != nil || days <= 0 {
		return
	}
	if err = uc.repo.ClearAlertsBefore(time.Now().AddDate(0, 0, -days)); err != nil {
		uc.log.Warn("failed to clear expired alerts", slog.Any("err", err))
	}
}

// clearState 清除某条规则的运行时状态，调用前需持有锁
func (uc *AlertUsecase) clearState(ruleID uint) {
	prefix := fmt.Sprintf("%d:", ruleID)
	for key := range uc.hits {
		if strings.HasPrefix(key, prefix) {
			delete(uc.hits, key)
		}
	}
	for key := range uc.fired {
		if strings.HasPrefix(key, prefix) {
			delete(uc.fired, key)
		}
	}
}

func (uc *AlertUsecase) buildMessage(rule *AlertRule, metric *AlertMetric) string {
	label := uc.metricLabel(rule.Type, metric.Target)

	switch rule.Type {
	case AlertTypeService:
		return uc.t.Get("service %s is not running", metric.Target)
	case AlertTypeProject:
		return uc.t.Get("project %s is not running", metric.Target)
	case AlertTypeContainer:
		return uc.t.Get("container %s is not running", metric.Target)
	case AlertTypeApp:
		return uc.t.Get("app %s is not running", metric.Target)
	case AlertTypeDatabase:
		return uc.t.Get("database server %s is unreachable", metric.Target)
	case AlertTypeCertExpire:
		return uc.t.Get("certificate %s expires in %s days", metric.Target, uc.formatValue(rule.Type, metric.Value))
	case AlertTypeWebsiteExpire:
		return uc.t.Get("website %s expires in %s days", metric.Target, uc.formatValue(rule.Type, metric.Value))
	}

	return uc.t.Get("%s is %s, %s threshold %s", label, uc.formatValue(rule.Type, metric.Value), uc.operatorLabel(rule.Operator), uc.formatValue(rule.Type, rule.Threshold))
}

func (uc *AlertUsecase) metricLabel(typ, target string) string {
	var label string
	switch typ {
	case AlertTypeCPU:
		label = uc.t.Get("CPU usage")
	case AlertTypeMemory:
		label = uc.t.Get("memory usage")
	case AlertTypeSwap:
		label = uc.t.Get("swap usage")
	case AlertTypeLoad1:
		label = uc.t.Get("1 minute load")
	case AlertTypeLoad5:
		label = uc.t.Get("5 minutes load")
	case AlertTypeLoad15:
		label = uc.t.Get("15 minutes load")
	case AlertTypeDisk:
		label = uc.t.Get("disk usage")
	case AlertTypeDiskInode:
		label = uc.t.Get("disk inode usage")
	case AlertTypeDiskRead:
		label = uc.t.Get("disk read speed")
	case AlertTypeDiskWrite:
		label = uc.t.Get("disk write speed")
	case AlertTypeNetIn:
		label = uc.t.Get("network download speed")
	case AlertTypeNetOut:
		label = uc.t.Get("network upload speed")
	case AlertTypeWebsite5xx:
		label = uc.t.Get("website 5xx responses this hour")
	case AlertTypeWebsiteError:
		label = uc.t.Get("website error rate this hour")
	case AlertTypeService:
		label = uc.t.Get("service status")
	case AlertTypeProject:
		label = uc.t.Get("project status")
	case AlertTypeContainer:
		label = uc.t.Get("container status")
	case AlertTypeApp:
		label = uc.t.Get("app status")
	case AlertTypeDatabase:
		label = uc.t.Get("database server status")
	case AlertTypeCertExpire:
		label = uc.t.Get("certificate expiry")
	case AlertTypeWebsiteExpire:
		label = uc.t.Get("website expiry")
	default:
		label = typ
	}

	if target != "" {
		return fmt.Sprintf("%s (%s)", label, target)
	}

	return label
}

func (uc *AlertUsecase) operatorLabel(operator string) string {
	switch operator {
	case AlertOperatorGTE:
		return uc.t.Get("greater than or equal to")
	case AlertOperatorLT:
		return uc.t.Get("less than")
	case AlertOperatorLTE:
		return uc.t.Get("less than or equal to")
	default:
		return uc.t.Get("greater than")
	}
}

func (uc *AlertUsecase) formatValue(typ string, value float64) string {
	switch typ {
	case AlertTypeCPU, AlertTypeMemory, AlertTypeSwap, AlertTypeDisk, AlertTypeDiskInode, AlertTypeWebsiteError:
		return fmt.Sprintf("%.2f%%", value)
	case AlertTypeNetIn, AlertTypeNetOut, AlertTypeDiskRead, AlertTypeDiskWrite:
		return fmt.Sprintf("%.2f MB/s", value)
	case AlertTypeCertExpire, AlertTypeWebsiteExpire, AlertTypeWebsite5xx:
		return fmt.Sprintf("%.0f", value)
	}

	if slices.Contains(statusAlertTypes, typ) {
		return uc.t.Get("not running")
	}

	return fmt.Sprintf("%.2f", value)
}

// normalizeRule 补齐规则的默认值，状态类规则固定为「不在运行」
func normalizeRule(rule *AlertRule) {
	if slices.Contains(statusAlertTypes, rule.Type) {
		rule.Operator = AlertOperatorGTE
		rule.Threshold = 1
	}
	if rule.Duration < 1 {
		rule.Duration = 1
	}
}

func matchThreshold(rule *AlertRule, value float64) bool {
	switch rule.Operator {
	case AlertOperatorGTE:
		return value >= rule.Threshold
	case AlertOperatorLT:
		return value < rule.Threshold
	case AlertOperatorLTE:
		return value <= rule.Threshold
	default:
		return value > rule.Threshold
	}
}

// rateOf 依据上次累计值计算每秒增量，首次采集速率为 0
func rateOf(prev ioSnapshot, in, out uint64, now time.Time) ioSnapshot {
	snap := ioSnapshot{in: in, out: out, at: now}
	if prev.at.IsZero() {
		return snap
	}

	elapsed := now.Sub(prev.at).Seconds()
	if elapsed < 1 {
		elapsed = 1
	}
	if in >= prev.in {
		snap.inRate = uint64(float64(in-prev.in) / elapsed)
	}
	if out >= prev.out {
		snap.outRate = uint64(float64(out-prev.out) / elapsed)
	}

	return snap
}

func stateKey(ruleID uint, target string) string {
	return fmt.Sprintf("%d:%s", ruleID, target)
}
