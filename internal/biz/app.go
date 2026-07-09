package biz

import (
	"time"

	"github.com/acepanel/panel/v3/pkg/api"
	"github.com/acepanel/panel/v3/pkg/types"
)

type App struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Slug      string    `gorm:"not null;default:'';unique" json:"slug"`
	Channel   string    `gorm:"not null;default:''" json:"channel"`
	Version   string    `gorm:"not null;default:''" json:"version"`
	Show      bool      `gorm:"not null;default:false" json:"show"`
	ShowOrder int       `gorm:"not null;default:0" json:"show_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AppRepo interface {
	Categories() []types.LV
	All() api.Apps
	Get(slug string) (*api.App, error)
	UpdateExist(slug string) bool
	Installed() ([]*App, error)
	GetInstalled(slug string) (*App, error)
	GetInstalledAll(query string, cond ...string) ([]*App, error)
	GetHomeShow() ([]map[string]string, error)
	IsInstalled(query string, cond ...any) (bool, error)
	Install(channel, slug string) error
	UnInstall(slug string) error
	Update(slug string) error
	UpdateShow(slug string, show bool) error
	UpdateOrder(slugs []string) error
}

type AppUsecase struct {
	repo AppRepo
}

func NewAppUsecase(repo AppRepo) *AppUsecase {
	return &AppUsecase{repo: repo}
}

func (uc *AppUsecase) Categories() []types.LV {
	return uc.repo.Categories()
}

func (uc *AppUsecase) All() api.Apps {
	return uc.repo.All()
}

func (uc *AppUsecase) Get(slug string) (*api.App, error) {
	return uc.repo.Get(slug)
}

func (uc *AppUsecase) UpdateExist(slug string) bool {
	return uc.repo.UpdateExist(slug)
}

func (uc *AppUsecase) Installed() ([]*App, error) {
	return uc.repo.Installed()
}

func (uc *AppUsecase) GetInstalled(slug string) (*App, error) {
	return uc.repo.GetInstalled(slug)
}

func (uc *AppUsecase) GetInstalledAll(query string, cond ...string) ([]*App, error) {
	return uc.repo.GetInstalledAll(query, cond...)
}

func (uc *AppUsecase) GetHomeShow() ([]map[string]string, error) {
	return uc.repo.GetHomeShow()
}

func (uc *AppUsecase) IsInstalled(query string, cond ...any) (bool, error) {
	return uc.repo.IsInstalled(query, cond...)
}

func (uc *AppUsecase) Install(channel, slug string) error {
	return uc.repo.Install(channel, slug)
}

func (uc *AppUsecase) UnInstall(slug string) error {
	return uc.repo.UnInstall(slug)
}

func (uc *AppUsecase) Update(slug string) error {
	return uc.repo.Update(slug)
}

func (uc *AppUsecase) UpdateShow(slug string, show bool) error {
	return uc.repo.UpdateShow(slug, show)
}

func (uc *AppUsecase) UpdateOrder(slugs []string) error {
	return uc.repo.UpdateOrder(slugs)
}
