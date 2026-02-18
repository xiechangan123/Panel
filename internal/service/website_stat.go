package service

import (
	"fmt"
	"net/http"
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

// queryTotals 查询指定日期范围的汇总数据
// 今日数据从内存聚合器获取（避免与 DB 重复计算），历史数据从 DB 查询
func (s *WebsiteStatService) queryTotals(start, end string, sites []string, today string) (*statTotals, error) {
	includeToday := start <= today && today <= end

	var total statTotals

	// DB 查询：排除今天（今天的数据从内存获取）
	dbEnd := end
	if includeToday {
		yesterday := time.Now().AddDate(0, 0, -1).Format(time.DateOnly)
		dbEnd = yesterday
	}

	if dbEnd >= start {
		dbStats, err := s.statRepo.ListByDateRange(start, dbEnd, sites)
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
	}

	// 今天的数据从内存获取
	if includeToday {
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

	if date == today {
		// 今天的小时数据从内存获取
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
	} else {
		// 历史数据从 DB 查询
		dbSeries, err := s.statRepo.HourlySeries(date, sites)
		if err != nil {
			return nil, err
		}
		for _, item := range dbSeries {
			h, _ := strconv.Atoi(item.Key)
			hourMap[h] = item
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

	// DB 查询排除今天
	dbEnd := end
	if includeToday {
		yesterday := time.Now().AddDate(0, 0, -1).Format(time.DateOnly)
		dbEnd = yesterday
	}

	dateMap := make(map[string]*biz.WebsiteStatSeries)
	if dbEnd >= start {
		dbSeries, err := s.statRepo.DailySeries(start, dbEnd, sites)
		if err != nil {
			return nil, err
		}
		for _, item := range dbSeries {
			dateMap[item.Key] = item
		}
	}

	// 今天的数据从内存获取
	if includeToday {
		liveStats := s.aggregator.SiteStats()
		siteSet := make(map[string]struct{}, len(sites))
		for _, name := range sites {
			siteSet[name] = struct{}{}
		}

		todayData := &biz.WebsiteStatSeries{Key: today}
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
		dateMap[today] = todayData
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

// Realtime 实时流量/RPS
func (s *WebsiteStatService) Realtime(w http.ResponseWriter, r *http.Request) {
	rt := s.aggregator.Realtime()
	Success(w, rt)
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
	Success(w, chix.M{
		"days": days,
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
