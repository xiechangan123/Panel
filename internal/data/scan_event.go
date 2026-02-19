package data

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/spf13/cast"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/acepanel/panel/internal/biz"
)

type scanEventRepo struct {
	db      *gorm.DB
	setting biz.SettingRepo
}

// NewScanEventRepo 创建扫描事件数据访问实例
func NewScanEventRepo(db *gorm.DB, setting biz.SettingRepo) biz.ScanEventRepo {
	return &scanEventRepo{
		db:      db,
		setting: setting,
	}
}

func (r scanEventRepo) Upsert(events []*biz.ScanEvent) error {
	if len(events) == 0 {
		return nil
	}

	const batchSize = 100
	for i := 0; i < len(events); i += batchSize {
		end := min(i+batchSize, len(events))
		if err := r.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "source_ip"}, {Name: "port"}, {Name: "protocol"}, {Name: "date"}},
			DoUpdates: clause.Assignments(map[string]any{"count": gorm.Expr("count + ?", gorm.Expr("excluded.count")), "last_seen": gorm.Expr("excluded.last_seen")}),
		}).Create(events[i:end]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r scanEventRepo) List(start, end, sourceIP string, port uint, page, limit uint) ([]*biz.ScanEvent, uint, error) {
	var total int64
	var items []*biz.ScanEvent

	tx := r.db.Model(&biz.ScanEvent{}).Where("date BETWEEN ? AND ?", start, end)
	if sourceIP != "" {
		tx = tx.Where("source_ip LIKE ?", "%"+sourceIP+"%")
	}
	if port > 0 {
		tx = tx.Where("port = ?", port)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := tx.Order("last_seen DESC").Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, uint(total), nil
}

func (r scanEventRepo) Summary(start, end string) (*biz.ScanSummary, error) {
	var summary biz.ScanSummary
	err := r.db.Model(&biz.ScanEvent{}).
		Where("date BETWEEN ? AND ?", start, end).
		Select("COALESCE(SUM(count), 0) as total_count, COUNT(DISTINCT source_ip) as unique_ips, COUNT(DISTINCT port || '-' || protocol) as unique_ports").
		Scan(&summary).Error
	return &summary, err
}

func (r scanEventRepo) Trend(start, end string) ([]*biz.ScanDayTrend, error) {
	var trends []*biz.ScanDayTrend
	err := r.db.Model(&biz.ScanEvent{}).
		Where("date BETWEEN ? AND ?", start, end).
		Select("date, COALESCE(SUM(count), 0) as total_count, COUNT(DISTINCT source_ip) as unique_ips").
		Group("date").
		Order("date ASC").
		Scan(&trends).Error
	return trends, err
}

func (r scanEventRepo) TopSourceIPs(start, end string, limit uint) ([]*biz.ScanSourceRank, error) {
	var ranks []*biz.ScanSourceRank
	err := r.db.Model(&biz.ScanEvent{}).
		Where("date BETWEEN ? AND ?", start, end).
		Select("source_ip, COALESCE(SUM(count), 0) as total_count, COUNT(DISTINCT port || '-' || protocol) as port_count, MAX(last_seen) as last_seen").
		Group("source_ip").
		Order("total_count DESC").
		Limit(int(limit)).
		Scan(&ranks).Error
	for _, rank := range ranks {
		rank.LastSeen = r.parseTimeStr(rank.LastSeen)
	}
	return ranks, err
}

func (r scanEventRepo) TopPorts(start, end string, limit uint) ([]*biz.ScanPortRank, error) {
	var ranks []*biz.ScanPortRank
	err := r.db.Model(&biz.ScanEvent{}).
		Where("date BETWEEN ? AND ?", start, end).
		Select("port, protocol, COALESCE(SUM(count), 0) as total_count, COUNT(DISTINCT source_ip) as ip_count").
		Group("port, protocol").
		Order("total_count DESC").
		Limit(int(limit)).
		Scan(&ranks).Error
	return ranks, err
}

func (r scanEventRepo) ClearBefore(date string) error {
	return r.db.Where("date < ?", date).Delete(&biz.ScanEvent{}).Error
}

func (r scanEventRepo) GetSetting() (*biz.ScanSetting, error) {
	enabled, err := r.setting.GetBool(biz.SettingKeyScanAware)
	if err != nil {
		return nil, err
	}
	days, err := r.setting.GetInt(biz.SettingKeyScanAwareDays, 30)
	if err != nil {
		return nil, err
	}

	interfacesStr, err := r.setting.Get(biz.SettingKeyScanAwareInterfaces)
	if err != nil {
		return nil, err
	}

	var interfaces []string
	if interfacesStr != "" {
		_ = json.Unmarshal([]byte(interfacesStr), &interfaces)
	}

	return &biz.ScanSetting{
		Enabled:    enabled,
		Days:       uint(days),
		Interfaces: interfaces,
	}, nil
}

func (r scanEventRepo) UpdateSetting(setting *biz.ScanSetting) error {
	if err := r.setting.Set(biz.SettingKeyScanAware, cast.ToString(setting.Enabled)); err != nil {
		return err
	}
	if err := r.setting.Set(biz.SettingKeyScanAwareDays, cast.ToString(setting.Days)); err != nil {
		return err
	}

	interfacesJSON, err := json.Marshal(setting.Interfaces)
	if err != nil {
		return err
	}
	return r.setting.Set(biz.SettingKeyScanAwareInterfaces, string(interfacesJSON))
}

func (r scanEventRepo) Clear() error {
	return r.db.Where("1 = 1").Delete(&biz.ScanEvent{}).Error
}

// parseTimeStr 解析 Go time.String() 格式并转为 RFC3339
func (r scanEventRepo) parseTimeStr(s string) string {
	if idx := strings.Index(s, " m="); idx > 0 {
		s = s[:idx]
	}
	if t, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", s); err == nil {
		return t.Format(time.RFC3339)
	}
	return s
}
