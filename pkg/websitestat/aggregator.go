package websitestat

import (
	"sync"
	"sync/atomic"
	"time"
)

// 详细统计每站每天最大条目数
const detailMaxKeys = 100000

// Aggregator 内存聚合器，将日志条目聚合为站点统计快照
type Aggregator struct {
	mu    sync.Mutex
	sites map[string]*siteDay
	date  string // 当前日期 YYYY-MM-DD

	// 错误日志缓冲
	errBuf []*ErrorEntry

	// 实时统计（滑动窗口 60 秒）
	rtSlots [60]rtSlot
	rtSec   int64
}

type hourBucket struct {
	pv, requests, errors, spiders, bandwidth uint64
	ips                                      map[string]struct{}
	uvs                                      map[string]struct{}
}

type siteDay struct {
	pv, requests, errors, spiders, bandwidth uint64
	ips                                      map[string]struct{}
	uvs                                      map[string]struct{}
	hours                                    [24]*hourBucket

	// 详细统计（增量计数器，DrainDetailStats 后清空）
	spiderCounts map[string]uint64       // spider_name → requests
	clientCounts map[string]*clientCount // "browser|os" → count
	ipCounts     map[string]*ipCount     // ip → count
	uriCounts    map[string]*uriCount    // uri → count
}

type clientCount struct {
	requests uint64
}

type ipCount struct {
	requests  uint64
	bandwidth uint64
}

type uriCount struct {
	requests  uint64
	bandwidth uint64
	errors    uint64
}

type rtSlot struct {
	bandwidth uint64
	requests  uint64
}

// NewAggregator 创建聚合器实例
func NewAggregator() *Aggregator {
	return &Aggregator{
		sites: make(map[string]*siteDay),
		date:  time.Now().Format(time.DateOnly),
	}
}

func newSiteDay() *siteDay {
	return &siteDay{
		ips:          make(map[string]struct{}),
		uvs:          make(map[string]struct{}),
		spiderCounts: make(map[string]uint64),
		clientCounts: make(map[string]*clientCount),
		ipCounts:     make(map[string]*ipCount),
		uriCounts:    make(map[string]*uriCount),
	}
}

// Record 记录一条访问日志
func (a *Aggregator) Record(entry *LogEntry) {
	now := time.Now()
	today := now.Format(time.DateOnly)
	hour := now.Hour()

	a.mu.Lock()

	// 日期切换，重置所有数据
	if today != a.date {
		a.sites = make(map[string]*siteDay)
		a.date = today
	}

	sd, ok := a.sites[entry.Site]
	if !ok {
		sd = newSiteDay()
		a.sites[entry.Site] = sd
	}

	// 确保小时桶已初始化
	hb := sd.hours[hour]
	if hb == nil {
		hb = &hourBucket{
			ips: make(map[string]struct{}),
			uvs: make(map[string]struct{}),
		}
		sd.hours[hour] = hb
	}

	// 更新每日总计
	sd.requests++
	sd.bandwidth += entry.Bytes

	// 更新小时桶
	hb.requests++
	hb.bandwidth += entry.Bytes

	// UV: IP+UA 去重（每日 + 每小时）
	uvKey := entry.IP + "|" + entry.UA
	sd.uvs[uvKey] = struct{}{}
	hb.uvs[uvKey] = struct{}{}

	// IP 去重（每日 + 每小时）
	sd.ips[entry.IP] = struct{}{}
	hb.ips[entry.IP] = struct{}{}

	// PV 判定
	if IsPageView(entry) {
		sd.pv++
		hb.pv++
	}

	isErr := entry.Status >= 400 && entry.Status < 600

	// 蜘蛛检测
	spiderName := SpiderName(entry.UA)
	if spiderName != "" {
		sd.spiders++
		hb.spiders++
		// 蜘蛛详细统计
		sd.spiderCounts[spiderName]++
	} else {
		// 非蜘蛛请求才统计客户端（浏览器/OS）
		browser, os := ParseUA(entry.UA)
		clientKey := browser + "|" + os
		if cc, exists := sd.clientCounts[clientKey]; exists {
			cc.requests++
		} else if len(sd.clientCounts) < detailMaxKeys {
			sd.clientCounts[clientKey] = &clientCount{requests: 1}
		}
	}

	// IP 详细统计
	if ic, exists := sd.ipCounts[entry.IP]; exists {
		ic.requests++
		ic.bandwidth += entry.Bytes
	} else if len(sd.ipCounts) < detailMaxKeys {
		sd.ipCounts[entry.IP] = &ipCount{requests: 1, bandwidth: entry.Bytes}
	}

	// URI 详细统计
	if uc, exists := sd.uriCounts[entry.URI]; exists {
		uc.requests++
		uc.bandwidth += entry.Bytes
		if isErr {
			uc.errors++
		}
	} else if len(sd.uriCounts) < detailMaxKeys {
		uc := &uriCount{requests: 1, bandwidth: entry.Bytes}
		if isErr {
			uc.errors = 1
		}
		sd.uriCounts[entry.URI] = uc
	}

	// 错误计数
	if isErr {
		sd.errors++
		hb.errors++
		a.errBuf = append(a.errBuf, &ErrorEntry{
			Site:   entry.Site,
			URI:    entry.URI,
			Method: entry.Method,
			IP:     entry.IP,
			UA:     entry.UA,
			Body:   entry.Body,
			Status: entry.Status,
		})
	}

	a.mu.Unlock()

	// 实时统计（原子操作，无需持有 mu）
	sec := now.Unix()
	idx := sec % 60
	curSec := atomic.LoadInt64(&a.rtSec)
	if sec != curSec {
		if atomic.CompareAndSwapInt64(&a.rtSec, curSec, sec) {
			// 清零当前槽
			atomic.StoreUint64(&a.rtSlots[idx].bandwidth, 0)
			atomic.StoreUint64(&a.rtSlots[idx].requests, 0)
		}
	}
	atomic.AddUint64(&a.rtSlots[idx].bandwidth, entry.Bytes)
	atomic.AddUint64(&a.rtSlots[idx].requests, 1)
}

// Snapshot 返回当前日期和各站点快照（不重置数据）
func (a *Aggregator) Snapshot() (string, map[string]*SiteSnapshot) {
	a.mu.Lock()
	defer a.mu.Unlock()

	date := a.date
	result := make(map[string]*SiteSnapshot, len(a.sites))
	for name, sd := range a.sites {
		snap := &SiteSnapshot{
			PV:        sd.pv,
			UV:        uint64(len(sd.uvs)),
			IP:        uint64(len(sd.ips)),
			Bandwidth: sd.bandwidth,
			Requests:  sd.requests,
			Errors:    sd.errors,
			Spiders:   sd.spiders,
		}

		// 填充小时快照
		for h := 0; h < 24; h++ {
			if hb := sd.hours[h]; hb != nil {
				snap.Hours[h] = &HourSnapshot{
					PV:        hb.pv,
					UV:        uint64(len(hb.uvs)),
					IP:        uint64(len(hb.ips)),
					Bandwidth: hb.bandwidth,
					Requests:  hb.requests,
					Errors:    hb.errors,
					Spiders:   hb.spiders,
				}
			}
		}

		result[name] = snap
	}

	return date, result
}

// SiteStats 返回各站点当前统计（不重置）
func (a *Aggregator) SiteStats() map[string]*SiteSnapshot {
	_, stats := a.Snapshot()
	return stats
}

// DrainErrors 取出并清空错误缓冲
func (a *Aggregator) DrainErrors() []*ErrorEntry {
	a.mu.Lock()
	defer a.mu.Unlock()

	errors := a.errBuf
	a.errBuf = nil
	return errors
}

// DrainDetailStats 导出并清空详细统计增量（蜘蛛/客户端/IP/URI）
func (a *Aggregator) DrainDetailStats() (string, map[string]*SiteDetailSnapshot) {
	a.mu.Lock()
	defer a.mu.Unlock()

	date := a.date
	result := make(map[string]*SiteDetailSnapshot, len(a.sites))

	for name, sd := range a.sites {
		snap := &SiteDetailSnapshot{
			Spiders: sd.spiderCounts,
			Clients: make(map[string]*ClientCount, len(sd.clientCounts)),
			IPs:     make(map[string]*IPCount, len(sd.ipCounts)),
			URIs:    make(map[string]*URICount, len(sd.uriCounts)),
		}

		for k, v := range sd.clientCounts {
			snap.Clients[k] = &ClientCount{Requests: v.requests}
		}
		for k, v := range sd.ipCounts {
			snap.IPs[k] = &IPCount{Requests: v.requests, Bandwidth: v.bandwidth}
		}
		for k, v := range sd.uriCounts {
			snap.URIs[k] = &URICount{Requests: v.requests, Bandwidth: v.bandwidth, Errors: v.errors}
		}

		result[name] = snap

		// 清空详细统计计数器
		sd.spiderCounts = make(map[string]uint64)
		sd.clientCounts = make(map[string]*clientCount)
		sd.ipCounts = make(map[string]*ipCount)
		sd.uriCounts = make(map[string]*uriCount)
	}

	return date, result
}

// Realtime 返回最近一段时间的实时流量和 RPS
func (a *Aggregator) Realtime() RealtimeStats {
	lastSec := atomic.LoadInt64(&a.rtSec)
	if lastSec == 0 {
		return RealtimeStats{}
	}

	now := time.Now().Unix()
	curIdx := now % 60

	var totalBw, totalReq uint64
	var count int64
	// 统计最近 5 秒（排除当前秒）
	for i := int64(1); i <= 5; i++ {
		idx := (curIdx - i + 60) % 60
		sec := now - i
		// 只统计最近 60 秒内的有效数据
		if sec >= lastSec-60 && sec <= lastSec {
			totalBw += atomic.LoadUint64(&a.rtSlots[idx].bandwidth)
			totalReq += atomic.LoadUint64(&a.rtSlots[idx].requests)
			count++
		}
	}

	if count == 0 {
		return RealtimeStats{}
	}

	return RealtimeStats{
		Bandwidth: float64(totalBw) / float64(count),
		RPS:       float64(totalReq) / float64(count),
	}
}
