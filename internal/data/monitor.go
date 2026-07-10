package data

import (
	"time"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
)

type monitorRepo struct {
	db *gorm.DB
}

func NewMonitorRepo(i do.Injector) (biz.MonitorRepo, error) {
	return &monitorRepo{
		db: do.MustInvoke[*gorm.DB](i),
	}, nil
}

func (r monitorRepo) Clear() error {
	return r.db.Where("1 = 1").Delete(&biz.Monitor{}).Error
}

func (r monitorRepo) List(start, end time.Time) ([]*biz.Monitor, error) {
	monitors := make([]*biz.Monitor, 0)
	if err := r.db.Where("created_at BETWEEN ? AND ?", start, end).Find(&monitors).Error; err != nil {
		return nil, err
	}

	return monitors, nil
}
