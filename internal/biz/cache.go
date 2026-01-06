package biz

import "time"

type CacheKey string

const (
	CacheKeyCategories  CacheKey = "categories"
	CacheKeyApps        CacheKey = "apps"
	CacheKeyEnvironment CacheKey = "environment"
	CacheKeyRewrites    CacheKey = "rewrites"
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
	UpdateRewrites() error
}
