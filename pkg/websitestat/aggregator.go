package websitestat

import (
	"sync"
	"sync/atomic"
	"time"
)

// Aggregator 内存聚合器，将日志条目聚合为站点统计快照
type Aggregator struct {
	mu    sync.Mutex
	sites map[string]*siteDay
	date  string // 当前日期 YYYY-MM-DD

	// 错误日志缓冲
	errBuf []*ErrorEntry

	// 可配置上限
	ErrBufMaxSize int  // 错误缓冲最大条目数，默认 10000
	UVMaxKeys     int  // 每站每天 UV 去重 set 上限，默认 1000000
	IPMaxKeys     int  // 每站每天 IP 去重 set 上限，默认 500000
	DetailMaxKeys int  // 详细统计（蜘蛛/客户端/IP/URI）每站每天最大条目数，默认 100000
	BodyEnabled   bool // 是否记录错误请求体，默认 true

	// 实时统计（滑动窗口 60 秒）
	rtSlots [60]rtSlot
	rtSec   int64
}

type hourBucket struct {
	pv, requests, errors, spiders, bandwidth uint64
	ips                                      map[string]struct{}
	uvs                                      map[string]struct{}

	// drain 后记录 set 大小，用于计算增量
	lastUVCount, lastIPCount uint64
}

type siteDay struct {
	pv, requests, errors, spiders, bandwidth uint64
	ips                                      map[string]struct{}
	uvs                                      map[string]struct{}
	hours                                    [24]*hourBucket

	// drain 后记录 set 大小，用于计算增量
	lastUVCount, lastIPCount uint64

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
		sites:         make(map[string]*siteDay),
		date:          time.Now().Format(time.DateOnly),
		ErrBufMaxSize: 10000,
		UVMaxKeys:     1000000,
		IPMaxKeys:     500000,
		DetailMaxKeys: 100000,
		BodyEnabled:   true,
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

	// 锁外预计算 CPU 密集操作
	spiderName := SpiderName(entry.UA)
	var browser, os string
	if spiderName == "" {
		browser, os = ParseUA(entry.UA)
	}
	uvKey := entry.IP + "|" + entry.UA
	isErr := entry.Status >= 400 && entry.Status < 600
	isPV := IsPageView(entry)
	clientKey := browser + "|" + os

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

	// UV: IP+UA 去重（每日 + 每小时），超限后不再插入
	if len(sd.uvs) < a.UVMaxKeys {
		sd.uvs[uvKey] = struct{}{}
	}
	hb.uvs[uvKey] = struct{}{}

	// IP 去重（每日 + 每小时），超限后不再插入
	if len(sd.ips) < a.IPMaxKeys {
		sd.ips[entry.IP] = struct{}{}
	}
	hb.ips[entry.IP] = struct{}{}

	// PV 判定
	if isPV {
		sd.pv++
		hb.pv++
	}

	// 蜘蛛检测
	if spiderName != "" {
		sd.spiders++
		hb.spiders++
		// 蜘蛛详细统计
		sd.spiderCounts[spiderName]++
	} else {
		// 非蜘蛛请求才统计客户端（浏览器/OS）
		if cc, exists := sd.clientCounts[clientKey]; exists {
			cc.requests++
		} else if len(sd.clientCounts) < a.DetailMaxKeys {
			sd.clientCounts[clientKey] = &clientCount{requests: 1}
		}
	}

	// IP 详细统计
	if ic, exists := sd.ipCounts[entry.IP]; exists {
		ic.requests++
		ic.bandwidth += entry.Bytes
	} else if len(sd.ipCounts) < a.DetailMaxKeys {
		sd.ipCounts[entry.IP] = &ipCount{requests: 1, bandwidth: entry.Bytes}
	}

	// URI 详细统计
	if uc, exists := sd.uriCounts[entry.URI]; exists {
		uc.requests++
		uc.bandwidth += entry.Bytes
		if isErr {
			uc.errors++
		}
	} else if len(sd.uriCounts) < a.DetailMaxKeys {
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
		if len(a.errBuf) < a.ErrBufMaxSize {
			errEntry := &ErrorEntry{
				Site:   entry.Site,
				URI:    entry.URI,
				Method: entry.Method,
				IP:     entry.IP,
				UA:     entry.UA,
				Status: entry.Status,
			}
			if a.BodyEnabled {
				errEntry.Body = entry.Body
			}
			a.errBuf = append(a.errBuf, errEntry)
		}
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

// snapshotHour 从 hourBucket 生成增量快照
func snapshotHour(hb *hourBucket) *HourSnapshot {
	if hb == nil {
		return nil
	}
	return &HourSnapshot{
		PV:        hb.pv,
		UV:        uint64(len(hb.uvs)) - hb.lastUVCount,
		IP:        uint64(len(hb.ips)) - hb.lastIPCount,
		Bandwidth: hb.bandwidth,
		Requests:  hb.requests,
		Errors:    hb.errors,
		Spiders:   hb.spiders,
	}
}

// snapshotSite 从 siteDay 生成增量快照
func snapshotSite(sd *siteDay) *SiteSnapshot {
	snap := &SiteSnapshot{
		PV:        sd.pv,
		UV:        uint64(len(sd.uvs)) - sd.lastUVCount,
		IP:        uint64(len(sd.ips)) - sd.lastIPCount,
		Bandwidth: sd.bandwidth,
		Requests:  sd.requests,
		Errors:    sd.errors,
		Spiders:   sd.spiders,
	}
	for h := 0; h < 24; h++ {
		snap.Hours[h] = snapshotHour(sd.hours[h])
	}
	return snap
}

// SiteStats 返回各站点自上次刷新以来的未刷新增量（用于实时展示叠加到 DB 数据上）
func (a *Aggregator) SiteStats() map[string]*SiteSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()

	result := make(map[string]*SiteSnapshot, len(a.sites))
	for name, sd := range a.sites {
		result[name] = snapshotSite(sd)
	}
	return result
}

// DrainSnapshot 返回自上次 drain 以来的增量快照，并重置计数器
// 用于将增量写入数据库（配合累加 upsert）
func (a *Aggregator) DrainSnapshot() (string, map[string]*SiteSnapshot) {
	a.mu.Lock()
	defer a.mu.Unlock()

	date := a.date
	result := make(map[string]*SiteSnapshot, len(a.sites))

	for name, sd := range a.sites {
		result[name] = snapshotSite(sd)

		// 重置可加计数器
		sd.pv, sd.requests, sd.errors, sd.spiders, sd.bandwidth = 0, 0, 0, 0, 0
		sd.lastUVCount = uint64(len(sd.uvs))
		sd.lastIPCount = uint64(len(sd.ips))

		// 重置小时桶
		for _, hb := range sd.hours {
			if hb == nil {
				continue
			}
			hb.pv, hb.requests, hb.errors, hb.spiders, hb.bandwidth = 0, 0, 0, 0, 0
			hb.lastUVCount = uint64(len(hb.uvs))
			hb.lastIPCount = uint64(len(hb.ips))
		}
	}

	return date, result
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
