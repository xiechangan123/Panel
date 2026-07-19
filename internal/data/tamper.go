package data

import (
	"time"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
)

type tamperRepo struct {
	db    *gorm.DB // 主库存规则
	logDB *gorm.DB // 独立库存拦截日志(高频写入)
}

// NewTamperRepo 创建防篡改数据访问实例
func NewTamperRepo(i do.Injector) (biz.TamperRepo, error) {
	db := do.MustInvoke[*gorm.DB](i)

	logDB, err := openDB("tamper")
	if err != nil {
		return nil, err
	}
	if err = logDB.AutoMigrate(&biz.TamperLog{}); err != nil {
		return nil, err
	}

	return &tamperRepo{db: db, logDB: logDB}, nil
}

func (r *tamperRepo) ListRules() ([]*biz.TamperRule, error) {
	var rules []*biz.TamperRule
	err := r.db.Order("id desc").Find(&rules).Error
	return rules, err
}

func (r *tamperRepo) GetRule(id uint) (*biz.TamperRule, error) {
	rule := new(biz.TamperRule)
	if err := r.db.First(rule, id).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

func (r *tamperRepo) GetRuleByName(name string) (*biz.TamperRule, error) {
	rule := new(biz.TamperRule)
	if err := r.db.Where("name = ?", name).First(rule).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

func (r *tamperRepo) CreateRule(rule *biz.TamperRule) error {
	return r.db.Create(rule).Error
}

func (r *tamperRepo) UpdateRule(rule *biz.TamperRule) error {
	return r.db.Model(&biz.TamperRule{}).Where("id = ?", rule.ID).Select("*").Updates(rule).Error
}

func (r *tamperRepo) DeleteRule(id uint) error {
	return r.db.Delete(&biz.TamperRule{}, id).Error
}

func (r *tamperRepo) AddLogs(logs []*biz.TamperLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.logDB.CreateInBatches(logs, 100).Error
}

func (r *tamperRepo) ListLogs(page, limit uint) ([]*biz.TamperLog, int64, error) {
	var logs []*biz.TamperLog
	var total int64
	if err := r.logDB.Model(&biz.TamperLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.logDB.Order("id desc").Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&logs).Error
	return logs, total, err
}

func (r *tamperRepo) ClearLogs() error {
	return r.logDB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&biz.TamperLog{}).Error
}

func (r *tamperRepo) ClearLogsBefore(t time.Time) error {
	return r.logDB.Where("created_at < ?", t).Delete(&biz.TamperLog{}).Error
}
