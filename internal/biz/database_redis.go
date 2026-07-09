package biz

import (
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/db"
)

type DatabaseRedisRepo interface {
	Databases(req *request.DatabaseRedisDatabases) (int, error)
	Data(req *request.DatabaseRedisData) ([]db.RedisKV, int, error)
	KeyGet(req *request.DatabaseRedisKeyGet) (*db.RedisKV, error)
	KeySet(req *request.DatabaseRedisKeySet) error
	KeyDelete(req *request.DatabaseRedisKeyDelete) error
	KeyTTL(req *request.DatabaseRedisKeyTTL) error
	KeyRename(req *request.DatabaseRedisKeyRename) error
	Clear(req *request.DatabaseRedisClear) error
}

// DatabaseRedisUsecase Redis 业务用例
type DatabaseRedisUsecase struct {
	repo DatabaseRedisRepo
}

func NewDatabaseRedisUsecase(repo DatabaseRedisRepo) *DatabaseRedisUsecase {
	return &DatabaseRedisUsecase{repo: repo}
}

func (uc *DatabaseRedisUsecase) Databases(req *request.DatabaseRedisDatabases) (int, error) {
	return uc.repo.Databases(req)
}

func (uc *DatabaseRedisUsecase) Data(req *request.DatabaseRedisData) ([]db.RedisKV, int, error) {
	return uc.repo.Data(req)
}

func (uc *DatabaseRedisUsecase) KeyGet(req *request.DatabaseRedisKeyGet) (*db.RedisKV, error) {
	return uc.repo.KeyGet(req)
}

func (uc *DatabaseRedisUsecase) KeySet(req *request.DatabaseRedisKeySet) error {
	return uc.repo.KeySet(req)
}

func (uc *DatabaseRedisUsecase) KeyDelete(req *request.DatabaseRedisKeyDelete) error {
	return uc.repo.KeyDelete(req)
}

func (uc *DatabaseRedisUsecase) KeyTTL(req *request.DatabaseRedisKeyTTL) error {
	return uc.repo.KeyTTL(req)
}

func (uc *DatabaseRedisUsecase) KeyRename(req *request.DatabaseRedisKeyRename) error {
	return uc.repo.KeyRename(req)
}

func (uc *DatabaseRedisUsecase) Clear(req *request.DatabaseRedisClear) error {
	return uc.repo.Clear(req)
}
