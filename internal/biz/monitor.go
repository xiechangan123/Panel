package biz

import (
	"time"

	"github.com/spf13/cast"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/types"
)

type Monitor struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	Info      types.CurrentInfo `gorm:"not null;default:'{}';serializer:json" json:"info"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type MonitorRepo interface {
	Clear() error
	List(start, end time.Time) ([]*Monitor, error)
}

type MonitorUsecase struct {
	repo    MonitorRepo
	setting SettingRepo
}

func NewMonitorUsecase(repo MonitorRepo, setting SettingRepo) *MonitorUsecase {
	return &MonitorUsecase{repo: repo, setting: setting}
}

func (uc *MonitorUsecase) GetSetting() (*request.MonitorSetting, error) {
	monitor, err := uc.setting.Get(SettingKeyMonitor)
	if err != nil {
		return nil, err
	}
	monitorDays, err := uc.setting.Get(SettingKeyMonitorDays)
	if err != nil {
		return nil, err
	}
	monitorInterval, err := uc.setting.GetInt(SettingKeyMonitorInterval, 1)
	if err != nil {
		return nil, err
	}

	setting := new(request.MonitorSetting)
	setting.Enabled = cast.ToBool(monitor)
	setting.Days = cast.ToUint(monitorDays)
	setting.Interval = uint(monitorInterval)

	return setting, nil
}

func (uc *MonitorUsecase) UpdateSetting(setting *request.MonitorSetting) error {
	if err := uc.setting.Set(SettingKeyMonitor, cast.ToString(setting.Enabled)); err != nil {
		return err
	}
	if err := uc.setting.Set(SettingKeyMonitorDays, cast.ToString(setting.Days)); err != nil {
		return err
	}
	if err := uc.setting.Set(SettingKeyMonitorInterval, cast.ToString(setting.Interval)); err != nil {
		return err
	}

	return nil
}

func (uc *MonitorUsecase) Clear() error {
	return uc.repo.Clear()
}

func (uc *MonitorUsecase) List(start, end time.Time) ([]*Monitor, error) {
	return uc.repo.List(start, end)
}
