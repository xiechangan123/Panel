package service

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/libtnb/chix"
	"github.com/samber/lo"

	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/websitestat"
)

type WebsiteStatService struct {
	setting     biz.SettingRepo
	statRepo    biz.WebsiteStatRepo
	websiteRepo biz.WebsiteRepo
	aggregator  *websitestat.Aggregator
}

func NewWebsiteStatService(setting biz.SettingRepo, statRepo biz.WebsiteStatRepo, websiteRepo biz.WebsiteRepo, aggregator *websitestat.Aggregator) *WebsiteStatService {
	return &WebsiteStatService{
		setting:     setting,
		statRepo:    statRepo,
		websiteRepo: websiteRepo,
		aggregator:  aggregator,
	}
}

// Overview 概览数据（汇总 + 时间序列 + 对比 + 站点列表）
func (s *WebsiteStatService) Overview(w http.ResponseWriter, r *http.Request) {
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	if start == "" || end == "" {
		today := time.Now().Format(time.DateOnly)
		start = today
		end = today
	}

	// 解析站点过滤
	var sites []string
	if sitesParam := r.URL.Query().Get("sites"); sitesParam != "" {
		sites = lo.Filter(strings.Split(sitesParam, ","), func(s string, _ int) bool {
			return strings.TrimSpace(s) != ""
		})
		sites = lo.Map(sites, func(s string, _ int) string {
			return strings.TrimSpace(s)
		})
	}

	// 计算对比周期
	startDate, _ := time.Parse(time.DateOnly, start)
	endDate, _ := time.Parse(time.DateOnly, end)
	duration := int(endDate.Sub(startDate).Hours()/24) + 1
	prevEnd := startDate.AddDate(0, 0, -1).Format(time.DateOnly)
	prevStart := startDate.AddDate(0, 0, -duration).Format(time.DateOnly)

	isSingleDay := start == end
	today := time.Now().Format(time.DateOnly)

	// 查询当前周期汇总
	current, err := s.queryTotals(start, end, sites, today)
	if err != nil {
		ErrorSystem(w)
		return
	}

	// 查询对比周期汇总
	previous, err := s.queryTotals(prevStart, prevEnd, sites, today)
	if err != nil {
		ErrorSystem(w)
		return
	}

	// 查询时间序列
	var series []*biz.WebsiteStatSeries
	if isSingleDay {
		series, err = s.queryHourlySeries(start, sites, today)
	} else {
		series, err = s.queryDailySeries(start, end, sites, today)
	}
	if err != nil {
		ErrorSystem(w)
		return
	}

	// 查询对比周期时间序列
	var prevSeries []*biz.WebsiteStatSeries
	prevIsSingleDay := prevStart == prevEnd
	if prevIsSingleDay {
		prevSeries, err = s.queryHourlySeries(prevStart, sites, today)
	} else {
		prevSeries, err = s.queryDailySeries(prevStart, prevEnd, sites, today)
	}
	if err != nil {
		ErrorSystem(w)
		return
	}

	// 获取所有网站列表（用于站点选择器）
	websites, _, err := s.websiteRepo.List("all", 1, 10000)
	if err != nil {
		ErrorSystem(w)
		return
	}

	siteList := lo.Map(websites, func(ws *biz.Website, _ int) chix.M {
		return chix.M{"id": ws.ID, "name": ws.Name}
	})

	Success(w, chix.M{
		"current":         current,
		"previous":        previous,
		"series":          series,
		"previous_series": prevSeries,
		"sites":           siteList,
	})
}

// Realtime 实时流量/RPS
func (s *WebsiteStatService) Realtime(w http.ResponseWriter, r *http.Request) {
	rt := s.aggregator.Realtime()
	Success(w, rt)
}

// SiteStats 网站维度汇总（每站 PV/UV/IP/带宽/请求/错误/蜘蛛）
func (s *WebsiteStatService) SiteStats(w http.ResponseWriter, r *http.Request) {
	start, end, sites := s.parseDateSites(r)
	today := time.Now().Format(time.DateOnly)
	includeToday := start <= today && today <= end

	siteMap := make(map[string]*biz.WebsiteStatSiteItem)

	// DB 查询全部日期范围（含今日）
	dbItems, err := s.statRepo.ListSiteStats(start, end, sites)
	if err != nil {
		ErrorSystem(w)
		return
	}
	for _, item := range dbItems {
		siteMap[item.Site] = item
	}

	// 叠加今日未刷新的增量
	if includeToday {
		liveStats := s.aggregator.SiteStats()
		siteSet := make(map[string]struct{}, len(sites))
		for _, name := range sites {
			siteSet[name] = struct{}{}
		}

		for name, snap := range liveStats {
			if len(sites) > 0 {
				if _, ok := siteSet[name]; !ok {
					continue
				}
			}
			if existing, ok := siteMap[name]; ok {
				existing.PV += snap.PV
				existing.UV += snap.UV
				existing.IP += snap.IP
				existing.Bandwidth += snap.Bandwidth
				existing.Requests += snap.Requests
				existing.Errors += snap.Errors
				existing.Spiders += snap.Spiders
			} else {
				siteMap[name] = &biz.WebsiteStatSiteItem{
					Site:      name,
					PV:        snap.PV,
					UV:        snap.UV,
					IP:        snap.IP,
					Bandwidth: snap.Bandwidth,
					Requests:  snap.Requests,
					Errors:    snap.Errors,
					Spiders:   snap.Spiders,
				}
			}
		}
	}

	// 转换为切片
	items := make([]*biz.WebsiteStatSiteItem, 0, len(siteMap))
	for _, item := range siteMap {
		items = append(items, item)
	}

	Success(w, chix.M{
		"items": items,
	})
}

// SpiderStats 蜘蛛统计排名
func (s *WebsiteStatService) SpiderStats(w http.ResponseWriter, r *http.Request) {
	start, end, sites := s.parseDateSites(r)

	items, err := s.statRepo.TopSpiders(start, end, sites, 50)
	if err != nil {
		ErrorSystem(w)
		return
	}

	// 计算总请求数和百分比
	var total uint64
	for _, item := range items {
		total += item.Requests
	}
	if total > 0 {
		for _, item := range items {
			item.Percent = float64(item.Requests) / float64(total) * 100
		}
	}

	Success(w, chix.M{
		"items": items,
		"total": total,
	})
}

// ClientStats 客户端统计
func (s *WebsiteStatService) ClientStats(w http.ResponseWriter, r *http.Request) {
	start, end, sites := s.parseDateSites(r)

	items, err := s.statRepo.TopClients(start, end, sites, 100)
	if err != nil {
		ErrorSystem(w)
		return
	}

	// 按浏览器聚合
	browserMap := make(map[string]uint64)
	osMap := make(map[string]uint64)
	for _, item := range items {
		browserMap[item.Browser] += item.Requests
		osMap[item.OS] += item.Requests
	}

	type rankItem struct {
		Name     string `json:"name"`
		Requests uint64 `json:"requests"`
	}
	browsers := make([]rankItem, 0, len(browserMap))
	for name, reqs := range browserMap {
		browsers = append(browsers, rankItem{Name: name, Requests: reqs})
	}
	oss := make([]rankItem, 0, len(osMap))
	for name, reqs := range osMap {
		oss = append(oss, rankItem{Name: name, Requests: reqs})
	}

	// 按请求数排序
	slices.SortFunc(browsers, func(a, b rankItem) int { return cmp.Compare(b.Requests, a.Requests) })
	slices.SortFunc(oss, func(a, b rankItem) int { return cmp.Compare(b.Requests, a.Requests) })

	Success(w, chix.M{
		"items":    items,
		"browsers": browsers,
		"os":       oss,
	})
}

// IPStats IP 统计（分页）
func (s *WebsiteStatService) IPStats(w http.ResponseWriter, r *http.Request) {
	start, end, sites := s.parseDateSites(r)

	page, _ := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
	limit, _ := strconv.ParseUint(r.URL.Query().Get("limit"), 10, 64)
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 50
	}

	items, total, err := s.statRepo.TopIPs(start, end, sites, uint(page), uint(limit))
	if err != nil {
		ErrorSystem(w)
		return
	}

	Success(w, chix.M{
		"items": items,
		"total": total,
	})
}

// URIStats URI 统计（分页）
func (s *WebsiteStatService) URIStats(w http.ResponseWriter, r *http.Request) {
	start, end, sites := s.parseDateSites(r)

	page, _ := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
	limit, _ := strconv.ParseUint(r.URL.Query().Get("limit"), 10, 64)
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 50
	}

	items, total, err := s.statRepo.TopURIs(start, end, sites, uint(page), uint(limit))
	if err != nil {
		ErrorSystem(w)
		return
	}

	Success(w, chix.M{
		"items": items,
		"total": total,
	})
}

// ErrorStats 错误日志（分页 + 状态码过滤）
func (s *WebsiteStatService) ErrorStats(w http.ResponseWriter, r *http.Request) {
	start, end, sites := s.parseDateSites(r)

	page, _ := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
	limit, _ := strconv.ParseUint(r.URL.Query().Get("limit"), 10, 64)
	status, _ := strconv.Atoi(r.URL.Query().Get("status"))
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 50
	}

	items, total, err := s.statRepo.ListErrors(start, end, sites, status, uint(page), uint(limit))
	if err != nil {
		ErrorSystem(w)
		return
	}

	Success(w, chix.M{
		"items": items,
		"total": total,
	})
}

// Clear 清空所有统计数据
func (s *WebsiteStatService) Clear(w http.ResponseWriter, r *http.Request) {
	if err := s.statRepo.Clear(); err != nil {
		ErrorSystem(w)
		return
	}
	Success(w, nil)
}

// GetSetting 获取统计设置
func (s *WebsiteStatService) GetSetting(w http.ResponseWriter, r *http.Request) {
	days, _ := s.setting.GetInt(biz.SettingKeyWebsiteStatDays, 30)
	errBufMax, _ := s.setting.GetInt(biz.SettingKeyWebsiteStatErrBufMax, 10000)
	uvMaxKeys, _ := s.setting.GetInt(biz.SettingKeyWebsiteStatUVMaxKeys, 1000000)
	ipMaxKeys, _ := s.setting.GetInt(biz.SettingKeyWebsiteStatIPMaxKeys, 500000)
	detailMaxKeys, _ := s.setting.GetInt(biz.SettingKeyWebsiteStatDetailMaxKeys, 100000)
	bodyEnabled, _ := s.setting.GetBool(biz.SettingKeyWebsiteStatBodyEnabled, true)
	Success(w, chix.M{
		"days":            days,
		"err_buf_max":     errBufMax,
		"uv_max_keys":     uvMaxKeys,
		"ip_max_keys":     ipMaxKeys,
		"detail_max_keys": detailMaxKeys,
		"body_enabled":    bodyEnabled,
	})
}

// UpdateSetting 更新统计设置
func (s *WebsiteStatService) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	req, err := Bind[request.WebsiteStatSetting](r)
	if err != nil {
		Error(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}

	if err = s.setting.Set(biz.SettingKeyWebsiteStatDays, fmt.Sprintf("%d", req.Days)); err != nil {
		Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	if req.ErrBufMax > 0 {
		_ = s.setting.Set(biz.SettingKeyWebsiteStatErrBufMax, fmt.Sprintf("%d", req.ErrBufMax))
	}
	if req.UVMaxKeys > 0 {
		_ = s.setting.Set(biz.SettingKeyWebsiteStatUVMaxKeys, fmt.Sprintf("%d", req.UVMaxKeys))
	}
	if req.IPMaxKeys > 0 {
		_ = s.setting.Set(biz.SettingKeyWebsiteStatIPMaxKeys, fmt.Sprintf("%d", req.IPMaxKeys))
	}
	if req.DetailMaxKeys > 0 {
		_ = s.setting.Set(biz.SettingKeyWebsiteStatDetailMaxKeys, fmt.Sprintf("%d", req.DetailMaxKeys))
	}
	_ = s.setting.Set(biz.SettingKeyWebsiteStatBodyEnabled, fmt.Sprintf("%t", req.BodyEnabled))

	Success(w, nil)
}

type statTotals struct {
	PV        uint64 `json:"pv"`
	UV        uint64 `json:"uv"`
	IP        uint64 `json:"ip"`
	Bandwidth uint64 `json:"bandwidth"`
	Requests  uint64 `json:"requests"`
	Errors    uint64 `json:"errors"`
	Spiders   uint64 `json:"spiders"`
}

// parseDateSites 解析公共查询参数 start, end, sites
func (s *WebsiteStatService) parseDateSites(r *http.Request) (start, end string, sites []string) {
	start = r.URL.Query().Get("start")
	end = r.URL.Query().Get("end")
	if start == "" || end == "" {
		today := time.Now().Format(time.DateOnly)
		start = today
		end = today
	}

	if sitesParam := r.URL.Query().Get("sites"); sitesParam != "" {
		sites = lo.Filter(strings.Split(sitesParam, ","), func(s string, _ int) bool {
			return strings.TrimSpace(s) != ""
		})
		sites = lo.Map(sites, func(s string, _ int) string {
			return strings.TrimSpace(s)
		})
	}
	return
}

// queryTotals 查询指定日期范围的汇总数据
// DB 包含全部日期（含今日已刷新数据），再叠加内存中未刷新的增量
func (s *WebsiteStatService) queryTotals(start, end string, sites []string, today string) (*statTotals, error) {
	var total statTotals

	// DB 查询全部日期范围（含今日）
	dbStats, err := s.statRepo.ListByDateRange(start, end, sites)
	if err != nil {
		return nil, err
	}
	for _, st := range dbStats {
		total.PV += st.PV
		total.UV += st.UV
		total.IP += st.IP
		total.Bandwidth += st.Bandwidth
		total.Requests += st.Requests
		total.Errors += st.Errors
		total.Spiders += st.Spiders
	}

	// 叠加今日未刷新的增量
	if start <= today && today <= end {
		s.mergeLiveTotals(&total, sites)
	}

	return &total, nil
}

// mergeLiveTotals 合并内存中今日实时数据到汇总
func (s *WebsiteStatService) mergeLiveTotals(total *statTotals, sites []string) {
	liveStats := s.aggregator.SiteStats()
	siteSet := make(map[string]struct{}, len(sites))
	for _, name := range sites {
		siteSet[name] = struct{}{}
	}

	for name, snap := range liveStats {
		if len(sites) > 0 {
			if _, ok := siteSet[name]; !ok {
				continue
			}
		}
		total.PV += snap.PV
		total.UV += snap.UV
		total.IP += snap.IP
		total.Bandwidth += snap.Bandwidth
		total.Requests += snap.Requests
		total.Errors += snap.Errors
		total.Spiders += snap.Spiders
	}
}

// queryHourlySeries 查询小时级时间序列
func (s *WebsiteStatService) queryHourlySeries(date string, sites []string, today string) ([]*biz.WebsiteStatSeries, error) {
	hourMap := make(map[int]*biz.WebsiteStatSeries, 24)

	// 从 DB 查询小时数据
	dbSeries, err := s.statRepo.HourlySeries(date, sites)
	if err != nil {
		return nil, err
	}
	for _, item := range dbSeries {
		h, _ := strconv.Atoi(item.Key)
		hourMap[h] = item
	}

	// 今天叠加内存中未刷新的增量
	if date == today {
		liveStats := s.aggregator.SiteStats()
		siteSet := make(map[string]struct{}, len(sites))
		for _, name := range sites {
			siteSet[name] = struct{}{}
		}

		for name, snap := range liveStats {
			if len(sites) > 0 {
				if _, ok := siteSet[name]; !ok {
					continue
				}
			}
			for h, hs := range snap.Hours {
				if hs == nil {
					continue
				}
				if existing, ok := hourMap[h]; ok {
					existing.PV += hs.PV
					existing.UV += hs.UV
					existing.IP += hs.IP
					existing.Bandwidth += hs.Bandwidth
					existing.Requests += hs.Requests
					existing.Errors += hs.Errors
					existing.Spiders += hs.Spiders
				} else {
					hourMap[h] = &biz.WebsiteStatSeries{
						Key:       strconv.Itoa(h),
						PV:        hs.PV,
						UV:        hs.UV,
						IP:        hs.IP,
						Bandwidth: hs.Bandwidth,
						Requests:  hs.Requests,
						Errors:    hs.Errors,
						Spiders:   hs.Spiders,
					}
				}
			}
		}
	}

	// 填充完整 24 小时
	result := make([]*biz.WebsiteStatSeries, 24)
	for h := range 24 {
		key := strconv.Itoa(h)
		if s, ok := hourMap[h]; ok {
			result[h] = s
		} else {
			result[h] = &biz.WebsiteStatSeries{Key: key}
		}
	}

	return result, nil
}

// queryDailySeries 查询日级时间序列
func (s *WebsiteStatService) queryDailySeries(start, end string, sites []string, today string) ([]*biz.WebsiteStatSeries, error) {
	includeToday := start <= today && today <= end

	// DB 查询全部日期范围（含今日）
	dateMap := make(map[string]*biz.WebsiteStatSeries)
	dbSeries, err := s.statRepo.DailySeries(start, end, sites)
	if err != nil {
		return nil, err
	}
	for _, item := range dbSeries {
		dateMap[item.Key] = item
	}

	// 今天叠加内存中未刷新的增量
	if includeToday {
		liveStats := s.aggregator.SiteStats()
		siteSet := make(map[string]struct{}, len(sites))
		for _, name := range sites {
			siteSet[name] = struct{}{}
		}

		todayData := dateMap[today]
		if todayData == nil {
			todayData = &biz.WebsiteStatSeries{Key: today}
			dateMap[today] = todayData
		}
		for name, snap := range liveStats {
			if len(sites) > 0 {
				if _, ok := siteSet[name]; !ok {
					continue
				}
			}
			todayData.PV += snap.PV
			todayData.UV += snap.UV
			todayData.IP += snap.IP
			todayData.Bandwidth += snap.Bandwidth
			todayData.Requests += snap.Requests
			todayData.Errors += snap.Errors
			todayData.Spiders += snap.Spiders
		}
	}

	// 填充日期范围内所有天
	startDate, _ := time.Parse(time.DateOnly, start)
	endDate, _ := time.Parse(time.DateOnly, end)
	var result []*biz.WebsiteStatSeries
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		key := d.Format(time.DateOnly)
		if s, ok := dateMap[key]; ok {
			result = append(result, s)
		} else {
			result = append(result, &biz.WebsiteStatSeries{Key: key})
		}
	}

	return result, nil
}
