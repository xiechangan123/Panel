package data

import (
	"encoding/json"
	"errors"
	"slices"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/pkg/api"
	"github.com/acepanel/panel/v3/pkg/apploader"
)

type cacheRepo struct {
	api *api.API
	db  *gorm.DB
}

func NewCacheRepo(db *gorm.DB) biz.CacheRepo {
	return &cacheRepo{
		api: api.NewAPI(app.Version, app.Locale),
		db:  db,
	}
}

func (r *cacheRepo) Get(key biz.CacheKey, defaultValue ...string) (string, error) {
	cache := new(biz.Cache)
	if err := r.db.Where("key = ?", key).First(cache).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
	}

	if cache.Value == "" && len(defaultValue) > 0 {
		return defaultValue[0], nil
	}

	return cache.Value, nil
}

func (r *cacheRepo) Set(key biz.CacheKey, value string) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&biz.Cache{Key: key, Value: value}).Error
}

func (r *cacheRepo) UpdateCategories() error {
	categories, err := r.api.Categories()
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(categories)
	if err != nil {
		return err
	}

	return r.Set(biz.CacheKeyCategories, string(encoded))
}

func (r *cacheRepo) UpdateApps() error {
	remote, err := r.api.Apps()
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

	return r.Set(biz.CacheKeyApps, string(encoded))
}

func (r *cacheRepo) UpdateEnvironments() error {
	environments, err := r.api.Environments()
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(environments)
	if err != nil {
		return err
	}

	return r.Set(biz.CacheKeyEnvironment, string(encoded))
}

func (r *cacheRepo) UpdateTemplates() error {
	templates, err := r.api.Templates()
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(templates)
	if err != nil {
		return err
	}

	return r.Set(biz.CacheKeyTemplates, string(encoded))
}
