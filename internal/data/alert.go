package data

import (
	"strings"
	"time"

	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/pkg/cert"
)

type alertRepo struct {
	db     *gorm.DB
	statDB *gorm.DB // 网站统计独立库
}

func NewAlertRepo(db *gorm.DB) (biz.AlertRepo, error) {
	statDB, err := openDB("stat")
	if err != nil {
		return nil, err
	}

	return &alertRepo{
		db:     db,
		statDB: statDB,
	}, nil
}

func (r *alertRepo) ListRules(page, limit uint) ([]*biz.AlertRule, int64, error) {
	rules := make([]*biz.AlertRule, 0)
	var total int64
	err := r.db.Model(&biz.AlertRule{}).Order("id desc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&rules).Error
	return rules, total, err
}

func (r *alertRepo) AllRules() ([]*biz.AlertRule, error) {
	rules := make([]*biz.AlertRule, 0)
	err := r.db.Order("id desc").Find(&rules).Error
	return rules, err
}

func (r *alertRepo) GetRule(id uint) (*biz.AlertRule, error) {
	rule := new(biz.AlertRule)
	if err := r.db.Where("id = ?", id).First(rule).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

func (r *alertRepo) CreateRule(rule *biz.AlertRule) error {
	return r.db.Create(rule).Error
}

func (r *alertRepo) UpdateRule(rule *biz.AlertRule) error {
	return r.db.Save(rule).Error
}

func (r *alertRepo) DeleteRule(id uint) error {
	return r.db.Where("id = ?", id).Delete(&biz.AlertRule{}).Error
}

func (r *alertRepo) AddAlert(alert *biz.Alert) error {
	return r.db.Create(alert).Error
}

func (r *alertRepo) ListAlerts(page, limit uint) ([]*biz.Alert, int64, error) {
	alerts := make([]*biz.Alert, 0)
	var total int64
	err := r.db.Model(&biz.Alert{}).Order("id desc").Count(&total).Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&alerts).Error
	return alerts, total, err
}

func (r *alertRepo) ClearAlerts() error {
	return r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&biz.Alert{}).Error
}

func (r *alertRepo) ClearAlertsBefore(t time.Time) error {
	return r.db.Where("created_at < ?", t).Delete(&biz.Alert{}).Error
}

func (r *alertRepo) CertExpiry() ([]*biz.AlertMetric, error) {
	certs := make([]*biz.Cert, 0)
	if err := r.db.Find(&certs).Error; err != nil {
		return nil, err
	}

	metrics := make([]*biz.AlertMetric, 0, len(certs))
	for _, item := range certs {
		if item.Cert == "" {
			continue
		}
		decode, err := cert.ParseCert([]byte(item.Cert))
		if err != nil {
			continue
		}
		// 一张证书一条指标，目标是逗号分隔的全部域名，规则可用其中任一域名匹配
		metrics = append(metrics, &biz.AlertMetric{
			Target: strings.Join(item.Domains, ","),
			Value:  time.Until(decode.NotAfter).Hours() / 24,
		})
	}

	return metrics, nil
}

// WebsiteHourStats 取各网站当前自然小时的请求统计
// 统计按自然小时聚合，整点归零，因此规则语义是「本小时累计」而非滚动一小时
func (r *alertRepo) WebsiteHourStats() ([]*biz.WebsiteHourStat, error) {
	now := time.Now()
	stats := make([]*biz.WebsiteStat, 0)
	if err := r.statDB.Where("date = ? AND hour = ?", now.Format(time.DateOnly), now.Hour()).Find(&stats).Error; err != nil {
		return nil, err
	}

	return lo.Map(stats, func(item *biz.WebsiteStat, _ int) *biz.WebsiteHourStat {
		return &biz.WebsiteHourStat{
			Site:      item.Site,
			Requests:  item.Requests,
			Errors:    item.Errors,
			Status5xx: item.Status5xx,
		}
	}), nil
}

func (r *alertRepo) ProjectNames() ([]string, error) {
	names := make([]string, 0)
	err := r.db.Model(&biz.Project{}).Pluck("name", &names).Error
	return names, err
}

// DatabaseServers 取数据库服务器列表
// 不复用 DatabaseServerRepo.List：它会串行探测每台服务器，探测由调用方按需并发进行
func (r *alertRepo) DatabaseServers() ([]*biz.DatabaseServer, error) {
	servers := make([]*biz.DatabaseServer, 0)
	err := r.db.Order("id desc").Find(&servers).Error
	return servers, err
}

func (r *alertRepo) WebsiteExpiry() ([]*biz.AlertMetric, error) {
	websites := make([]*biz.Website, 0)
	if err := r.db.Where("expire_at IS NOT NULL").Find(&websites).Error; err != nil {
		return nil, err
	}

	metrics := make([]*biz.AlertMetric, 0, len(websites))
	for _, item := range websites {
		if item.ExpireAt == nil {
			continue
		}
		metrics = append(metrics, &biz.AlertMetric{
			Target: item.Name,
			Value:  time.Until(*item.ExpireAt).Hours() / 24,
		})
	}

	return metrics, nil
}
