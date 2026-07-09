package biz

import "time"

type CacheKey string

const (
	CacheKeyCategories  CacheKey = "categories"
	CacheKeyApps        CacheKey = "apps"
	CacheKeyEnvironment CacheKey = "environment"
	CacheKeyTemplates   CacheKey = "templates"
)

type Cache struct {
	Key       CacheKey  `gorm:"primaryKey" json:"key"`
	Value     string    `gorm:"not null;default:''" json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CacheRepo interface {
	Get(key CacheKey, defaultValue ...string) (string, error)
	Set(key CacheKey, value string) error
	UpdateCategories() error
	UpdateApps() error
	UpdateEnvironments() error
	UpdateTemplates() error
}

type CacheUsecase struct {
	repo CacheRepo
}

func NewCacheUsecase(repo CacheRepo) *CacheUsecase {
	return &CacheUsecase{repo: repo}
}

func (uc *CacheUsecase) Get(key CacheKey, defaultValue ...string) (string, error) {
	return uc.repo.Get(key, defaultValue...)
}

func (uc *CacheUsecase) Set(key CacheKey, value string) error {
	return uc.repo.Set(key, value)
}

func (uc *CacheUsecase) UpdateCategories() error {
	return uc.repo.UpdateCategories()
}

func (uc *CacheUsecase) UpdateApps() error {
	return uc.repo.UpdateApps()
}

func (uc *CacheUsecase) UpdateEnvironments() error {
	return uc.repo.UpdateEnvironments()
}

func (uc *CacheUsecase) UpdateTemplates() error {
	return uc.repo.UpdateTemplates()
}
