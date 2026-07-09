package data

import (
	"time"

	"github.com/samber/do/v2"
	"github.com/spf13/cast"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/request"
)

type monitorRepo struct {
	db      *gorm.DB
	setting biz.SettingRepo
}

func NewMonitorRepo(i do.Injector) (biz.MonitorRepo, error) {
	return &monitorRepo{
		db:      do.MustInvoke[*gorm.DB](i),
		setting: do.MustInvoke[biz.SettingRepo](i),
	}, nil
}

func (r monitorRepo) GetSetting() (*request.MonitorSetting, error) {
	monitor, err := r.setting.Get(biz.SettingKeyMonitor)
	if err != nil {
		return nil, err
	}
	monitorDays, err := r.setting.Get(biz.SettingKeyMonitorDays)
	if err != nil {
		return nil, err
	}
	monitorInterval, err := r.setting.GetInt(biz.SettingKeyMonitorInterval, 1)
	if err != nil {
		return nil, err
	}

	setting := new(request.MonitorSetting)
	setting.Enabled = cast.ToBool(monitor)
	setting.Days = cast.ToUint(monitorDays)
	setting.Interval = uint(monitorInterval)

	return setting, nil
}

func (r monitorRepo) UpdateSetting(setting *request.MonitorSetting) error {
	if err := r.setting.Set(biz.SettingKeyMonitor, cast.ToString(setting.Enabled)); err != nil {
		return err
	}
	if err := r.setting.Set(biz.SettingKeyMonitorDays, cast.ToString(setting.Days)); err != nil {
		return err
	}
	if err := r.setting.Set(biz.SettingKeyMonitorInterval, cast.ToString(setting.Interval)); err != nil {
		return err
	}

	return nil
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
