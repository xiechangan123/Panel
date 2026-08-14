package data

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/migration"
)

type monitorRepo struct {
	db *gorm.DB
}

func NewMonitorRepo() (biz.MonitorRepo, error) {
	monitorDB, err := openDB("monitor")
	if err != nil {
		return nil, err
	}
	if err = gormigrate.New(monitorDB, nil, migration.MonitorMigrations).Migrate(); err != nil {
		return nil, err
	}

	return &monitorRepo{db: monitorDB}, nil
}

func (r *monitorRepo) Create(monitor *biz.Monitor) error {
	return r.db.Create(monitor).Error
}

func (r *monitorRepo) ClearBefore(t time.Time) error {
	return r.db.Where("created_at < ?", t).Delete(&biz.Monitor{}).Error
}

func (r *monitorRepo) Clear() error {
	return r.db.Where("1 = 1").Delete(&biz.Monitor{}).Error
}

func (r *monitorRepo) List(start, end time.Time) ([]*biz.Monitor, error) {
	monitors := make([]*biz.Monitor, 0)
	if err := r.db.Where("created_at BETWEEN ? AND ?", start, end).Find(&monitors).Error; err != nil {
		return nil, err
	}

	return monitors, nil
}

func (r *monitorRepo) VacuumDB() error {
	return vacuumDB(r.db)
}
