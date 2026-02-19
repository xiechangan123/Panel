package websitestat

import (
	"maps"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Aggregator 内存聚合器，将日志条目聚合为站点统计快照
type Aggregator struct {
	mu   sync.Mutex
	days map[string]map[string]*siteDay // date -> site -> stats

	// 错误日志缓冲
	errBuf []*ErrorEntry

	// 可配置上限
	ErrBufMaxSize int  // 错误缓冲最大条目数，默认 10000
	UVMaxKeys     int  // 每站每天 UV 去重 set 上限，默认 1000000
	IPMaxKeys     int  // 每站每天 IP 去重 set 上限，默认 500000
	DetailMaxKeys int  // 详细统计（蜘蛛/客户端/IP/URI）每站每天最大条目数，默认 100000
	BodyEnabled   bool // 是否记录错误请求体，默认 false

	// 实时统计（滑动窗口 60 秒）
	rtSlots [60]rtSlot
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
	sec       int64
	bandwidth uint64
	requests  uint64
}

// NewAggregator 创建聚合器实例
func NewAggregator() *Aggregator {
	return &Aggregator{
		days:          make(map[string]map[string]*siteDay),
		ErrBufMaxSize: 10000,
		UVMaxKeys:     1000000,
		IPMaxKeys:     500000,
		DetailMaxKeys: 100000,
		BodyEnabled:   false,
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

	sites, ok := a.days[today]
	if !ok {
		sites = make(map[string]*siteDay)
		a.days[today] = sites
	}

	sd, ok := sites[entry.Site]
	if !ok {
		sd = newSiteDay()
		sites[entry.Site] = sd
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
	if len(hb.uvs) < a.UVMaxKeys {
		hb.uvs[uvKey] = struct{}{}
	}

	// IP 去重（每日 + 每小时），超限后不再插入
	if len(sd.ips) < a.IPMaxKeys {
		sd.ips[entry.IP] = struct{}{}
	}
	if len(hb.ips) < a.IPMaxKeys {
		hb.ips[entry.IP] = struct{}{}
	}

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

	// 实时统计（无锁原子写入）
	a.addRealtime(now.Unix(), entry.Bytes)
}

// SiteStats 返回各站点自上次刷新以来的未刷新增量（用于实时展示叠加到 DB 数据上）
func (a *Aggregator) SiteStats() map[string]*SiteSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()

	today := time.Now().Format(time.DateOnly)
	sites := a.days[today]
	if len(sites) == 0 {
		return map[string]*SiteSnapshot{}
	}

	result := make(map[string]*SiteSnapshot, len(sites))
	for name, sd := range sites {
		result[name] = snapshotSite(sd)
	}
	return result
}

// DrainSnapshot 返回自上次 drain 以来的增量快照
// 返回 commit 函数，调用方写库成功后必须调用 commit 来消费已快照的增量
// 若写库失败不调用 commit，数据保留在内存中，下次 drain 会再次包含
func (a *Aggregator) DrainSnapshot() (snapshotsByDate map[string]map[string]*SiteSnapshot, commit func()) {
	a.mu.Lock()
	defer a.mu.Unlock()

	snapshotsByDate = make(map[string]map[string]*SiteSnapshot, len(a.days))
	for date, sites := range a.days {
		snapshots := make(map[string]*SiteSnapshot, len(sites))
		for name, sd := range sites {
			snapshots[name] = snapshotSite(sd)
		}
		snapshotsByDate[date] = snapshots
	}

	commit = func() {
		a.mu.Lock()
		defer a.mu.Unlock()

		for date, snapshots := range snapshotsByDate {
			sites, ok := a.days[date]
			if !ok {
				continue
			}
			for name, snap := range snapshots {
				sd, ok := sites[name]
				if !ok {
					continue
				}

				// 减去已快照的增量，保留快照后新到的数据
				sd.pv = saturatingSub(sd.pv, snap.PV)
				sd.requests = saturatingSub(sd.requests, snap.Requests)
				sd.errors = saturatingSub(sd.errors, snap.Errors)
				sd.spiders = saturatingSub(sd.spiders, snap.Spiders)
				sd.bandwidth = saturatingSub(sd.bandwidth, snap.Bandwidth)
				sd.lastUVCount = cappedAdd(sd.lastUVCount, snap.UV, uint64(len(sd.uvs)))
				sd.lastIPCount = cappedAdd(sd.lastIPCount, snap.IP, uint64(len(sd.ips)))

				for h, hs := range snap.Hours {
					if hs == nil {
						continue
					}
					hb := sd.hours[h]
					if hb == nil {
						continue
					}
					hb.pv = saturatingSub(hb.pv, hs.PV)
					hb.requests = saturatingSub(hb.requests, hs.Requests)
					hb.errors = saturatingSub(hb.errors, hs.Errors)
					hb.spiders = saturatingSub(hb.spiders, hs.Spiders)
					hb.bandwidth = saturatingSub(hb.bandwidth, hs.Bandwidth)
					hb.lastUVCount = cappedAdd(hb.lastUVCount, hs.UV, uint64(len(hb.uvs)))
					hb.lastIPCount = cappedAdd(hb.lastIPCount, hs.IP, uint64(len(hb.ips)))
				}
			}
		}

		a.gcDrainedPastDaysLocked()
	}

	return snapshotsByDate, commit
}

// DrainErrors 取出错误缓冲快照
// 返回 commit 函数，写库成功后调用以移除已消费的条目
func (a *Aggregator) DrainErrors() (entries []*ErrorEntry, commit func()) {
	a.mu.Lock()
	n := len(a.errBuf)
	entries = make([]*ErrorEntry, n)
	copy(entries, a.errBuf)
	a.mu.Unlock()

	commit = func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		// 只移除已快照的前 n 条，保留期间新增的
		if n >= len(a.errBuf) {
			a.errBuf = nil
		} else {
			a.errBuf = a.errBuf[n:]
		}
	}
	return
}

// Reset 清空所有内存聚合数据（配合 DB 清空使用）
func (a *Aggregator) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.days = make(map[string]map[string]*siteDay)
	a.errBuf = nil
}

// DrainDetailStats 导出详细统计增量（蜘蛛/客户端/IP/URI）
// 返回 commit 函数，调用方写库成功后必须调用 commit 来清空计数器
func (a *Aggregator) DrainDetailStats() (detailsByDate map[string]map[string]*SiteDetailSnapshot, commit func()) {
	a.mu.Lock()
	defer a.mu.Unlock()

	detailsByDate = make(map[string]map[string]*SiteDetailSnapshot, len(a.days))
	for date, sites := range a.days {
		details := make(map[string]*SiteDetailSnapshot, len(sites))
		for name, sd := range sites {
			// 复制 Spiders map，避免共享引用被后续写入污染
			spiders := make(map[string]uint64, len(sd.spiderCounts))
			maps.Copy(spiders, sd.spiderCounts)

			snap := &SiteDetailSnapshot{
				Spiders: spiders,
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

			details[name] = snap
		}
		detailsByDate[date] = details
	}

	commit = func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		for date, details := range detailsByDate {
			sites, ok := a.days[date]
			if !ok {
				continue
			}
			for name, snap := range details {
				sd, ok := sites[name]
				if !ok {
					continue
				}
				for spider, requests := range snap.Spiders {
					if cur := sd.spiderCounts[spider]; cur <= requests {
						delete(sd.spiderCounts, spider)
					} else {
						sd.spiderCounts[spider] = cur - requests
					}
				}
				for key, cc := range snap.Clients {
					if cur, exists := sd.clientCounts[key]; exists {
						if cur.requests <= cc.Requests {
							delete(sd.clientCounts, key)
						} else {
							cur.requests -= cc.Requests
						}
					}
				}
				for ip, ic := range snap.IPs {
					if cur, exists := sd.ipCounts[ip]; exists {
						cur.requests = saturatingSub(cur.requests, ic.Requests)
						cur.bandwidth = saturatingSub(cur.bandwidth, ic.Bandwidth)
						if cur.requests == 0 && cur.bandwidth == 0 {
							delete(sd.ipCounts, ip)
						}
					}
				}
				for uri, uc := range snap.URIs {
					if cur, exists := sd.uriCounts[uri]; exists {
						cur.requests = saturatingSub(cur.requests, uc.Requests)
						cur.bandwidth = saturatingSub(cur.bandwidth, uc.Bandwidth)
						cur.errors = saturatingSub(cur.errors, uc.Errors)
						if cur.requests == 0 && cur.bandwidth == 0 && cur.errors == 0 {
							delete(sd.uriCounts, uri)
						}
					}
				}
			}
		}
		a.gcDrainedPastDaysLocked()
	}

	return detailsByDate, commit
}

// Realtime 返回最近一段时间的实时流量和 RPS
func (a *Aggregator) Realtime() RealtimeStats {
	now := time.Now().Unix()

	var totalBw, totalReq uint64
	var count int64
	// 统计最近 5 秒（排除当前秒）
	for i := int64(1); i <= 5; i++ {
		sec := now - i
		idx := ((sec % 60) + 60) % 60
		// 校验槽位秒戳，只读取确实属于目标秒的数据
		if atomic.LoadInt64(&a.rtSlots[idx].sec) == sec {
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

// gcDrainedPastDaysLocked 清理已经完全刷盘的历史日期桶
func (a *Aggregator) gcDrainedPastDaysLocked() {
	today := time.Now().Format(time.DateOnly)
	for date, sites := range a.days {
		if date == today {
			continue
		}
		canDrop := true
		for _, sd := range sites {
			if hasPending(sd) {
				canDrop = false
				break
			}
		}
		if canDrop {
			delete(a.days, date)
		}
	}
}

// addRealtime 写入实时槽位，保证秒切换时不会因清零覆盖并发写入
func (a *Aggregator) addRealtime(sec int64, bytes uint64) {
	idx := sec % 60
	slot := &a.rtSlots[idx]
	initMark := -sec

	for {
		cur := atomic.LoadInt64(&slot.sec)
		if cur == sec {
			atomic.AddUint64(&slot.bandwidth, bytes)
			atomic.AddUint64(&slot.requests, 1)
			return
		}

		// 其他协程正在初始化当前秒槽位，等待其完成
		if cur == initMark || cur < 0 {
			runtime.Gosched()
			continue
		}

		// 抢到初始化权后，先清零，再发布秒戳，最后写入当前请求
		if atomic.CompareAndSwapInt64(&slot.sec, cur, initMark) {
			atomic.StoreUint64(&slot.bandwidth, 0)
			atomic.StoreUint64(&slot.requests, 0)
			atomic.StoreInt64(&slot.sec, sec)
			atomic.AddUint64(&slot.bandwidth, bytes)
			atomic.AddUint64(&slot.requests, 1)
			return
		}
	}
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
	for h := range 24 {
		snap.Hours[h] = snapshotHour(sd.hours[h])
	}
	return snap
}

// hasPending 判断站点是否仍有未落库增量
func hasPending(sd *siteDay) bool {
	if sd.pv > 0 || sd.requests > 0 || sd.errors > 0 || sd.spiders > 0 || sd.bandwidth > 0 {
		return true
	}
	for _, hb := range sd.hours {
		if hb == nil {
			continue
		}
		if hb.pv > 0 || hb.requests > 0 || hb.errors > 0 || hb.spiders > 0 || hb.bandwidth > 0 {
			return true
		}
	}
	return len(sd.spiderCounts) > 0 || len(sd.clientCounts) > 0 || len(sd.ipCounts) > 0 || len(sd.uriCounts) > 0
}

// saturatingSub 无符号减法（防止下溢）
func saturatingSub(v, sub uint64) uint64 {
	if v <= sub {
		return 0
	}
	return v - sub
}

// cappedAdd 累加后不超过 cap
func cappedAdd(v, add, maxVal uint64) uint64 {
	if v >= maxVal {
		return maxVal
	}
	if add > maxVal-v {
		return maxVal
	}
	return v + add
}
