package biz

import "time"

// WebsiteStat 网站统计（每站每天一行或每小时一行）
type WebsiteStat struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Site      string    `gorm:"not null;uniqueIndex:idx_wstat_unique" json:"site"`
	Date      string    `gorm:"not null;uniqueIndex:idx_wstat_unique;index" json:"date"`
	Hour      int       `gorm:"not null;uniqueIndex:idx_wstat_unique;default:-1" json:"hour"` // -1=每日汇总, 0-23=小时
	PV        uint64    `gorm:"not null;default:0" json:"pv"`
	UV        uint64    `gorm:"not null;default:0" json:"uv"`
	IP        uint64    `gorm:"not null;default:0" json:"ip"`
	Bandwidth uint64    `gorm:"not null;default:0" json:"bandwidth"`
	Requests  uint64    `gorm:"not null;default:0" json:"requests"`
	Errors    uint64    `gorm:"not null;default:0" json:"errors"`
	Spiders   uint64    `gorm:"not null;default:0" json:"spiders"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WebsiteErrorLog 网站错误日志（400-599 状态码详情）
type WebsiteErrorLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Site      string    `gorm:"not null;index:idx_werr_site_time" json:"site"`
	URI       string    `gorm:"not null" json:"uri"`
	Method    string    `gorm:"not null" json:"method"`
	Status    int       `gorm:"not null;index" json:"status"`
	IP        string    `gorm:"not null" json:"ip"`
	UA        string    `gorm:"not null" json:"ua"`
	Body      string    `gorm:"type:text" json:"body"`
	CreatedAt time.Time `gorm:"index:idx_werr_site_time" json:"created_at"`
}

// WebsiteStatSeries 时间序列数据点（用于 API 返回）
type WebsiteStatSeries struct {
	Key       string `json:"key"` // 小时 "0"-"23" 或日期 "2026-02-18"
	PV        uint64 `json:"pv"`
	UV        uint64 `json:"uv"`
	IP        uint64 `json:"ip"`
	Bandwidth uint64 `json:"bandwidth"`
	Requests  uint64 `json:"requests"`
	Errors    uint64 `json:"errors"`
	Spiders   uint64 `json:"spiders"`
}

// WebsiteStatRepo 网站统计数据访问接口
type WebsiteStatRepo interface {
	// Upsert 批量写入/更新统计数据（含每日汇总和小时数据）
	Upsert(stats []*WebsiteStat) error
	// ListByDateRange 按日期范围查询每站汇总（仅 hour=-1 的每日行）
	ListByDateRange(start, end string, sites []string) ([]*WebsiteStat, error)
	// DailySeries 按日分组的时间序列
	DailySeries(start, end string, sites []string) ([]*WebsiteStatSeries, error)
	// HourlySeries 按小时的时间序列（单天查询）
	HourlySeries(date string, sites []string) ([]*WebsiteStatSeries, error)
	// ClearBefore 清理过期数据
	ClearBefore(date string) error
	// InsertErrors 批量写入错误日志
	InsertErrors(errors []*WebsiteErrorLog) error
	// ClearErrorsBefore 清理过期错误日志
	ClearErrorsBefore(t time.Time) error
	// Clear 清空所有统计数据
	Clear() error
}
