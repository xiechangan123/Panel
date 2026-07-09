package biz

import (
	"time"

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
	GetSetting() (*request.MonitorSetting, error)
	UpdateSetting(setting *request.MonitorSetting) error
	Clear() error
	List(start, end time.Time) ([]*Monitor, error)
}

type MonitorUsecase struct {
	repo MonitorRepo
}

func NewMonitorUsecase(repo MonitorRepo) *MonitorUsecase {
	return &MonitorUsecase{repo: repo}
}

func (uc *MonitorUsecase) GetSetting() (*request.MonitorSetting, error) {
	return uc.repo.GetSetting()
}

func (uc *MonitorUsecase) UpdateSetting(setting *request.MonitorSetting) error {
	return uc.repo.UpdateSetting(setting)
}

func (uc *MonitorUsecase) Clear() error {
	return uc.repo.Clear()
}

func (uc *MonitorUsecase) List(start, end time.Time) ([]*Monitor, error) {
	return uc.repo.List(start, end)
}
