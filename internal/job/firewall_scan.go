package job

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/pkg/firewall/scan"
	"github.com/acepanel/panel/pkg/geoip"
)

// FirewallScan 防火墙扫描感知任务
type FirewallScan struct {
	log          *slog.Logger
	setting      biz.SettingRepo
	scanRepo     biz.ScanEventRepo
	scanner      *scan.Scanner
	geoIP        *geoip.GeoIP
	geoIPPath    string
	geoIPModTime time.Time
	buffer       map[string]*biz.ScanEvent // key: "ip:port:proto:date"
	mu           sync.Mutex
}

// NewFirewallScan 创建扫描感知任务
func NewFirewallScan(log *slog.Logger, setting biz.SettingRepo, scanRepo biz.ScanEventRepo) *FirewallScan {
	return &FirewallScan{
		log:      log,
		setting:  setting,
		scanRepo: scanRepo,
		buffer:   make(map[string]*biz.ScanEvent),
	}
}

func (r *FirewallScan) Run() {
	if app.Status != app.StatusNormal {
		return
	}

	enabled, err := r.setting.GetBool(biz.SettingKeyScanAware)
	if err != nil || !enabled {
		// 未启用时，确保 scanner 已停止
		r.stopScanner()
		return
	}

	// 确保 scanner 已启动
	r.ensureScanner()

	// 热更新 GeoIP
	r.geoIP, r.geoIPPath, r.geoIPModTime = refreshGeoIP(r.setting, r.geoIP, r.geoIPPath, r.geoIPModTime, r.log)

	// flush 缓冲到数据库
	r.flush()

	// 清理过期数据
	r.cleanup()
}

// ensureScanner 确保 scanner 正在运行
func (r *FirewallScan) ensureScanner() {
	if r.scanner != nil {
		return
	}

	if !scan.Supported() {
		return
	}

	setting, err := r.scanRepo.GetSetting()
	if err != nil {
		r.log.Warn("failed to get scan setting", slog.Any("err", err))
		return
	}

	scanner, err := scan.New(setting.Interfaces, r.log)
	if err != nil {
		r.log.Warn("failed to start eBPF scan detector", slog.Any("err", err))
		return
	}

	r.scanner = scanner

	// 启动后台事件聚合
	go r.aggregate()
}

// stopScanner 停止 scanner
func (r *FirewallScan) stopScanner() {
	if r.scanner == nil {
		return
	}
	_ = r.scanner.Close()
	r.scanner = nil
}

// aggregate 持续读取 eBPF 事件并聚合到内存缓冲
func (r *FirewallScan) aggregate() {
	events := r.scanner.Events()
	if events == nil {
		return
	}

	for evt := range events {
		date := evt.Timestamp.Format(time.DateOnly)
		key := fmt.Sprintf("%s:%d:%s:%s", evt.SourceIP, evt.Port, evt.Protocol, date)

		r.mu.Lock()
		if existing, ok := r.buffer[key]; ok {
			existing.Count++
			existing.LastSeen = evt.Timestamp
		} else {
			r.buffer[key] = &biz.ScanEvent{
				SourceIP:  evt.SourceIP,
				Port:      uint(evt.Port),
				Protocol:  evt.Protocol,
				Date:      date,
				Count:     1,
				FirstSeen: evt.Timestamp,
				LastSeen:  evt.Timestamp,
			}
		}
		r.mu.Unlock()
	}
}

// flush 将内存缓冲写入数据库
func (r *FirewallScan) flush() {
	r.mu.Lock()
	if len(r.buffer) == 0 {
		r.mu.Unlock()
		return
	}

	events := make([]*biz.ScanEvent, 0, len(r.buffer))
	for _, evt := range r.buffer {
		events = append(events, evt)
	}
	r.buffer = make(map[string]*biz.ScanEvent)
	r.mu.Unlock()

	// 解析 IP 地理位置
	if r.geoIP != nil {
		for _, evt := range events {
			geo := r.geoIP.Lookup(evt.SourceIP)
			evt.Country = geo.Country
			evt.Region = geo.Region
			evt.City = geo.City
			evt.District = geo.District
		}
	}

	if err := r.scanRepo.Upsert(events); err != nil {
		r.log.Warn("failed to upsert scan events", slog.Any("err", err))
	}
}

// cleanup 清理过期数据
func (r *FirewallScan) cleanup() {
	day, err := r.setting.GetInt(biz.SettingKeyScanAwareDays, 30)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -day).Format(time.DateOnly)
	if err = r.scanRepo.ClearBefore(cutoff); err != nil {
		r.log.Warn("failed to clear expired scan data", slog.Any("err", err))
	}
}
