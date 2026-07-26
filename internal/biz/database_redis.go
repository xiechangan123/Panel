package biz

import (
	"context"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/db"
)

type DatabaseRedisRepo interface {
	Databases(ctx context.Context, req *request.DatabaseRedisDatabases) (int, error)
	Data(ctx context.Context, req *request.DatabaseRedisData) ([]db.RedisKV, int, error)
	KeyGet(ctx context.Context, req *request.DatabaseRedisKeyGet) (*db.RedisKV, error)
	KeySet(ctx context.Context, req *request.DatabaseRedisKeySet) error
	KeyDelete(ctx context.Context, req *request.DatabaseRedisKeyDelete) error
	KeyTTL(ctx context.Context, req *request.DatabaseRedisKeyTTL) error
	KeyRename(ctx context.Context, req *request.DatabaseRedisKeyRename) error
	Clear(ctx context.Context, req *request.DatabaseRedisClear) error
}

// DatabaseRedisUsecase Redis 业务用例
type DatabaseRedisUsecase struct {
	repo DatabaseRedisRepo
}

func NewDatabaseRedisUsecase(repo DatabaseRedisRepo) *DatabaseRedisUsecase {
	return &DatabaseRedisUsecase{repo: repo}
}

func (uc *DatabaseRedisUsecase) Databases(ctx context.Context, req *request.DatabaseRedisDatabases) (int, error) {
	return uc.repo.Databases(ctx, req)
}

func (uc *DatabaseRedisUsecase) Data(ctx context.Context, req *request.DatabaseRedisData) ([]db.RedisKV, int, error) {
	return uc.repo.Data(ctx, req)
}

func (uc *DatabaseRedisUsecase) KeyGet(ctx context.Context, req *request.DatabaseRedisKeyGet) (*db.RedisKV, error) {
	return uc.repo.KeyGet(ctx, req)
}

func (uc *DatabaseRedisUsecase) KeySet(ctx context.Context, req *request.DatabaseRedisKeySet) error {
	return uc.repo.KeySet(ctx, req)
}

func (uc *DatabaseRedisUsecase) KeyDelete(ctx context.Context, req *request.DatabaseRedisKeyDelete) error {
	return uc.repo.KeyDelete(ctx, req)
}

func (uc *DatabaseRedisUsecase) KeyTTL(ctx context.Context, req *request.DatabaseRedisKeyTTL) error {
	return uc.repo.KeyTTL(ctx, req)
}

func (uc *DatabaseRedisUsecase) KeyRename(ctx context.Context, req *request.DatabaseRedisKeyRename) error {
	return uc.repo.KeyRename(ctx, req)
}

func (uc *DatabaseRedisUsecase) Clear(ctx context.Context, req *request.DatabaseRedisClear) error {
	return uc.repo.Clear(ctx, req)
}
