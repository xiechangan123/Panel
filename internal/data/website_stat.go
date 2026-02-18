package data

import (
	"fmt"
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

	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "site"}, {Name: "date"}, {Name: "hour"}},
		DoUpdates: clause.Assignments(map[string]any{
			"pv":         gorm.Expr("excluded.pv"),
			"uv":         gorm.Expr("excluded.uv"),
			"ip":         gorm.Expr("excluded.ip"),
			"bandwidth":  gorm.Expr("excluded.bandwidth"),
			"requests":   gorm.Expr("excluded.requests"),
			"errors":     gorm.Expr("excluded.errors"),
			"spiders":    gorm.Expr("excluded.spiders"),
			"updated_at": gorm.Expr("excluded.updated_at"),
		}),
	}).Create(stats).Error
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
		Select(fmt.Sprintf("CAST(hour AS TEXT) as key, COALESCE(SUM(pv), 0) as pv, COALESCE(SUM(uv), 0) as uv, COALESCE(SUM(ip), 0) as ip, COALESCE(SUM(bandwidth), 0) as bandwidth, COALESCE(SUM(requests), 0) as requests, COALESCE(SUM(errors), 0) as errors, COALESCE(SUM(spiders), 0) as spiders")).
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

	const batchSize = 100
	for i := 0; i < len(errors); i += batchSize {
		end := i + batchSize
		if end > len(errors) {
			end = len(errors)
		}
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
	return r.db.Where("1 = 1").Delete(&biz.WebsiteErrorLog{}).Error
}
