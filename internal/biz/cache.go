package biz

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/acepanel/panel/v3/pkg/api"
	"github.com/acepanel/panel/v3/pkg/apploader"
)

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
	FetchCategories() (*api.Categories, error)
	FetchApps() (*api.Apps, error)
	FetchEnvironments() (*api.Environments, error)
	FetchTemplates() (*api.Templates, error)
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
	categories, err := uc.repo.FetchCategories()
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(categories)
	if err != nil {
		return err
	}

	return uc.repo.Set(CacheKeyCategories, string(encoded))
}

func (uc *CacheUsecase) UpdateApps() error {
	remote, err := uc.repo.FetchApps()
	if err != nil {
		return err
	}

	// 去除本地不存在的应用
	*remote = slices.Clip(slices.DeleteFunc(*remote, func(item *api.App) bool {
		return !slices.Contains(apploader.Slugs(), item.Slug)
	}))

	encoded, err := json.Marshal(remote)
	if err != nil {
		return err
	}

	return uc.repo.Set(CacheKeyApps, string(encoded))
}

func (uc *CacheUsecase) UpdateEnvironments() error {
	environments, err := uc.repo.FetchEnvironments()
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(environments)
	if err != nil {
		return err
	}

	return uc.repo.Set(CacheKeyEnvironment, string(encoded))
}

func (uc *CacheUsecase) UpdateTemplates() error {
	templates, err := uc.repo.FetchTemplates()
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(templates)
	if err != nil {
		return err
	}

	return uc.repo.Set(CacheKeyTemplates, string(encoded))
}
