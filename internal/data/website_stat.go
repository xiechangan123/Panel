package data

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/acepanel/panel/internal/biz"
)

type websiteStatRepo struct {
	db *gorm.DB
}

// NewWebsiteStatRepo 创建网站统计数据访问实例
func NewWebsiteStatRepo(db *gorm.DB) biz.WebsiteStatRepo {
	return &websiteStatRepo{db: db}
}

func (r *websiteStatRepo) Upsert(stats []*biz.WebsiteStat) error {
	if len(stats) == 0 {
		return nil
	}

	return batchUpsert(r.db, stats, clause.OnConflict{
		Columns: []clause.Column{{Name: "site"}, {Name: "date"}, {Name: "hour"}},
		DoUpdates: clause.Assignments(map[string]any{
			"pv":         gorm.Expr("website_stats.pv + excluded.pv"),
			"uv":         gorm.Expr("website_stats.uv + excluded.uv"),
			"ip":         gorm.Expr("website_stats.ip + excluded.ip"),
			"bandwidth":  gorm.Expr("website_stats.bandwidth + excluded.bandwidth"),
			"requests":   gorm.Expr("website_stats.requests + excluded.requests"),
			"errors":     gorm.Expr("website_stats.errors + excluded.errors"),
			"spiders":    gorm.Expr("website_stats.spiders + excluded.spiders"),
			"updated_at": gorm.Expr("excluded.updated_at"),
		}),
	})
}

func (r *websiteStatRepo) ListByDateRange(start, end string, sites []string) ([]*biz.WebsiteStat, error) {
	var items []*biz.WebsiteStat
	q := r.db.Model(&biz.WebsiteStat{}).
		Select("site, SUM(pv) as pv, SUM(uv) as uv, SUM(ip) as ip, SUM(bandwidth) as bandwidth, SUM(requests) as requests, SUM(errors) as errors, SUM(spiders) as spiders").
		Where("date BETWEEN ? AND ? AND hour = -1", start, end)
	if len(sites) > 0 {
		q = q.Where("site IN ?", sites)
	}
	err := q.Group("site").Scan(&items).Error
	return items, err
}

func (r *websiteStatRepo) DailySeries(start, end string, sites []string) ([]*biz.WebsiteStatSeries, error) {
	var series []*biz.WebsiteStatSeries
	q := r.db.Model(&biz.WebsiteStat{}).
		Select("date as key, COALESCE(SUM(pv), 0) as pv, COALESCE(SUM(uv), 0) as uv, COALESCE(SUM(ip), 0) as ip, COALESCE(SUM(bandwidth), 0) as bandwidth, COALESCE(SUM(requests), 0) as requests, COALESCE(SUM(errors), 0) as errors, COALESCE(SUM(spiders), 0) as spiders").
		Where("date BETWEEN ? AND ? AND hour = -1", start, end)
	if len(sites) > 0 {
		q = q.Where("site IN ?", sites)
	}
	err := q.Group("date").Order("date ASC").Scan(&series).Error
	return series, err
}

func (r *websiteStatRepo) HourlySeries(date string, sites []string) ([]*biz.WebsiteStatSeries, error) {
	var series []*biz.WebsiteStatSeries
	q := r.db.Model(&biz.WebsiteStat{}).
		Select("CAST(hour AS TEXT) as key, COALESCE(SUM(pv), 0) as pv, COALESCE(SUM(uv), 0) as uv, COALESCE(SUM(ip), 0) as ip, COALESCE(SUM(bandwidth), 0) as bandwidth, COALESCE(SUM(requests), 0) as requests, COALESCE(SUM(errors), 0) as errors, COALESCE(SUM(spiders), 0) as spiders").
		Where("date = ? AND hour >= 0", date)
	if len(sites) > 0 {
		q = q.Where("site IN ?", sites)
	}
	err := q.Group("hour").Order("hour ASC").Scan(&series).Error
	return series, err
}

func (r *websiteStatRepo) ClearBefore(date string) error {
	return r.db.Where("date < ?", date).Delete(&biz.WebsiteStat{}).Error
}

func (r *websiteStatRepo) InsertErrors(errors []*biz.WebsiteErrorLog) error {
	if len(errors) == 0 {
		return nil
	}

	for i := 0; i < len(errors); i += upsertBatchSize {
		end := min(i+upsertBatchSize, len(errors))
		if err := r.db.Create(errors[i:end]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *websiteStatRepo) ClearErrorsBefore(t time.Time) error {
	return r.db.Where("created_at < ?", t).Delete(&biz.WebsiteErrorLog{}).Error
}

func (r *websiteStatRepo) Clear() error {
	if err := r.db.Where("1 = 1").Delete(&biz.WebsiteStat{}).Error; err != nil {
		return err
	}
	if err := r.db.Where("1 = 1").Delete(&biz.WebsiteErrorLog{}).Error; err != nil {
		return err
	}
	if err := r.db.Where("1 = 1").Delete(&biz.WebsiteStatSpider{}).Error; err != nil {
		return err
	}
	if err := r.db.Where("1 = 1").Delete(&biz.WebsiteStatClient{}).Error; err != nil {
		return err
	}
	if err := r.db.Where("1 = 1").Delete(&biz.WebsiteStatIP{}).Error; err != nil {
		return err
	}
	return r.db.Where("1 = 1").Delete(&biz.WebsiteStatURI{}).Error
}

// ========== 蜘蛛统计 ==========

func (r *websiteStatRepo) UpsertSpiders(stats []*biz.WebsiteStatSpider) error {
	if len(stats) == 0 {
		return nil
	}
	return batchUpsert(r.db, stats, clause.OnConflict{
		Columns: []clause.Column{{Name: "site"}, {Name: "date"}, {Name: "spider"}},
		DoUpdates: clause.Assignments(map[string]any{
			"requests":   gorm.Expr("website_stat_spiders.requests + excluded.requests"),
			"updated_at": gorm.Expr("excluded.updated_at"),
		}),
	})
}

func (r *websiteStatRepo) TopSpiders(start, end string, sites []string, limit uint) ([]*biz.WebsiteStatSpiderRank, error) {
	var items []*biz.WebsiteStatSpiderRank
	q := r.db.Model(&biz.WebsiteStatSpider{}).
		Select("spider, SUM(requests) as requests").
		Where("date BETWEEN ? AND ?", start, end)
	if len(sites) > 0 {
		q = q.Where("site IN ?", sites)
	}
	err := q.Group("spider").Order("requests DESC").Limit(int(limit)).Scan(&items).Error
	return items, err
}

func (r *websiteStatRepo) ClearSpidersBefore(date string) error {
	return r.db.Where("date < ?", date).Delete(&biz.WebsiteStatSpider{}).Error
}

// ========== 客户端统计 ==========

func (r *websiteStatRepo) UpsertClients(stats []*biz.WebsiteStatClient) error {
	if len(stats) == 0 {
		return nil
	}
	return batchUpsert(r.db, stats, clause.OnConflict{
		Columns: []clause.Column{{Name: "site"}, {Name: "date"}, {Name: "browser"}, {Name: "os"}},
		DoUpdates: clause.Assignments(map[string]any{
			"requests":   gorm.Expr("website_stat_clients.requests + excluded.requests"),
			"updated_at": gorm.Expr("excluded.updated_at"),
		}),
	})
}

func (r *websiteStatRepo) TopClients(start, end string, sites []string, limit uint) ([]*biz.WebsiteStatClientRank, error) {
	var items []*biz.WebsiteStatClientRank
	q := r.db.Model(&biz.WebsiteStatClient{}).
		Select("browser, os, SUM(requests) as requests").
		Where("date BETWEEN ? AND ?", start, end)
	if len(sites) > 0 {
		q = q.Where("site IN ?", sites)
	}
	err := q.Group("browser, os").Order("requests DESC").Limit(int(limit)).Scan(&items).Error
	return items, err
}

func (r *websiteStatRepo) ClearClientsBefore(date string) error {
	return r.db.Where("date < ?", date).Delete(&biz.WebsiteStatClient{}).Error
}

// ========== IP 统计 ==========

func (r *websiteStatRepo) UpsertIPs(stats []*biz.WebsiteStatIP) error {
	if len(stats) == 0 {
		return nil
	}
	return batchUpsert(r.db, stats, clause.OnConflict{
		Columns: []clause.Column{{Name: "site"}, {Name: "date"}, {Name: "ip"}},
		DoUpdates: clause.Assignments(map[string]any{
			"requests":   gorm.Expr("website_stat_ips.requests + excluded.requests"),
			"bandwidth":  gorm.Expr("website_stat_ips.bandwidth + excluded.bandwidth"),
			"updated_at": gorm.Expr("excluded.updated_at"),
		}),
	})
}

func (r *websiteStatRepo) TopIPs(start, end string, sites []string, page, limit uint) ([]*biz.WebsiteStatIPRank, uint, error) {
	var total int64
	q := r.db.Model(&biz.WebsiteStatIP{}).
		Where("date BETWEEN ? AND ?", start, end)
	if len(sites) > 0 {
		q = q.Where("site IN ?", sites)
	}

	// 计算唯一 IP 总数
	countQ := r.db.Table("(?) as sub",
		q.Select("ip").Group("ip"),
	)
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []*biz.WebsiteStatIPRank
	dataQ := r.db.Model(&biz.WebsiteStatIP{}).
		Select("ip, SUM(requests) as requests, SUM(bandwidth) as bandwidth").
		Where("date BETWEEN ? AND ?", start, end)
	if len(sites) > 0 {
		dataQ = dataQ.Where("site IN ?", sites)
	}
	offset := (page - 1) * limit
	err := dataQ.Group("ip").Order("requests DESC").Offset(int(offset)).Limit(int(limit)).Scan(&items).Error
	return items, uint(total), err
}

func (r *websiteStatRepo) ClearIPsBefore(date string) error {
	return r.db.Where("date < ?", date).Delete(&biz.WebsiteStatIP{}).Error
}

// ========== URI 统计 ==========

func (r *websiteStatRepo) UpsertURIs(stats []*biz.WebsiteStatURI) error {
	if len(stats) == 0 {
		return nil
	}
	return batchUpsert(r.db, stats, clause.OnConflict{
		Columns: []clause.Column{{Name: "site"}, {Name: "date"}, {Name: "uri"}},
		DoUpdates: clause.Assignments(map[string]any{
			"requests":   gorm.Expr("website_stat_uris.requests + excluded.requests"),
			"bandwidth":  gorm.Expr("website_stat_uris.bandwidth + excluded.bandwidth"),
			"errors":     gorm.Expr("website_stat_uris.errors + excluded.errors"),
			"updated_at": gorm.Expr("excluded.updated_at"),
		}),
	})
}

func (r *websiteStatRepo) TopURIs(start, end string, sites []string, page, limit uint) ([]*biz.WebsiteStatURIRank, uint, error) {
	var total int64
	q := r.db.Model(&biz.WebsiteStatURI{}).
		Where("date BETWEEN ? AND ?", start, end)
	if len(sites) > 0 {
		q = q.Where("site IN ?", sites)
	}

	// 计算唯一 URI 总数
	countQ := r.db.Table("(?) as sub",
		q.Select("uri").Group("uri"),
	)
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []*biz.WebsiteStatURIRank
	dataQ := r.db.Model(&biz.WebsiteStatURI{}).
		Select("uri, SUM(requests) as requests, SUM(bandwidth) as bandwidth, SUM(errors) as errors").
		Where("date BETWEEN ? AND ?", start, end)
	if len(sites) > 0 {
		dataQ = dataQ.Where("site IN ?", sites)
	}
	offset := (page - 1) * limit
	err := dataQ.Group("uri").Order("requests DESC").Offset(int(offset)).Limit(int(limit)).Scan(&items).Error
	return items, uint(total), err
}

func (r *websiteStatRepo) ClearURIsBefore(date string) error {
	return r.db.Where("date < ?", date).Delete(&biz.WebsiteStatURI{}).Error
}

// ========== 错误日志查询 ==========

func (r *websiteStatRepo) ListErrors(start, end string, sites []string, status int, page, limit uint) ([]*biz.WebsiteErrorLog, uint, error) {
	var total int64
	q := r.db.Model(&biz.WebsiteErrorLog{}).
		Where("created_at >= ? AND created_at < DATE(?, '+1 day')", start, end)
	if len(sites) > 0 {
		q = q.Where("site IN ?", sites)
	}
	if status > 0 {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []*biz.WebsiteErrorLog
	offset := (page - 1) * limit
	err := q.Order("created_at DESC").Offset(int(offset)).Limit(int(limit)).Find(&items).Error
	return items, uint(total), err
}

// ========== 网站维度汇总 ==========

func (r *websiteStatRepo) ListSiteStats(start, end string, sites []string) ([]*biz.WebsiteStatSiteItem, error) {
	var items []*biz.WebsiteStatSiteItem
	q := r.db.Model(&biz.WebsiteStat{}).
		Select("site, SUM(pv) as pv, SUM(uv) as uv, SUM(ip) as ip, SUM(bandwidth) as bandwidth, SUM(requests) as requests, SUM(errors) as errors, SUM(spiders) as spiders").
		Where("date BETWEEN ? AND ? AND hour = -1", start, end)
	if len(sites) > 0 {
		q = q.Where("site IN ?", sites)
	}
	err := q.Group("site").Order("requests DESC").Scan(&items).Error
	return items, err
}

// upsert 分批大小，避免超出 SQLite 变量数限制
const upsertBatchSize = 100

// batchUpsert 通用分批 upsert 辅助函数
func batchUpsert[T any](db *gorm.DB, items []T, conflict clause.OnConflict) error {
	for i := 0; i < len(items); i += upsertBatchSize {
		end := min(i+upsertBatchSize, len(items))
		if err := db.Clauses(conflict).Create(items[i:end]).Error; err != nil {
			return err
		}
	}
	return nil
}
