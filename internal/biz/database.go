package biz

import (
	"context"

	"github.com/acepanel/panel/v3/internal/request"
)

type DatabaseType string

const (
	DatabaseTypeMysql         DatabaseType = "mysql"
	DatabaseTypePostgresql    DatabaseType = "postgresql"
	DatabaseTypeMongoDB       DatabaseType = "mongodb"
	DatabaseTypeClickHouse    DatabaseType = "clickhouse"
	DatabaseTypeSQLite        DatabaseType = "sqlite"
	DatabaseTypeRedis         DatabaseType = "redis"
	DatabaseTypeElasticsearch DatabaseType = "elasticsearch"
)

type Database struct {
	Type     DatabaseType `json:"type"`
	Name     string       `json:"name"`
	Server   string       `json:"server"`
	ServerID uint         `json:"server_id"`
	Encoding string       `json:"encoding"`
	Comment  string       `json:"comment"`
}

type DatabaseRepo interface {
	List(page, limit uint, typ string) ([]*Database, int64, error)
	Create(ctx context.Context, req *request.DatabaseCreate) error
	Delete(ctx context.Context, serverID uint, name string) error
	Comment(req *request.DatabaseComment) error
}

// DatabaseUsecase 数据库业务用例
type DatabaseUsecase struct {
	repo DatabaseRepo
}

func NewDatabaseUsecase(repo DatabaseRepo) *DatabaseUsecase {
	return &DatabaseUsecase{repo: repo}
}

func (uc *DatabaseUsecase) List(page, limit uint, typ string) ([]*Database, int64, error) {
	return uc.repo.List(page, limit, typ)
}

func (uc *DatabaseUsecase) Create(ctx context.Context, req *request.DatabaseCreate) error {
	return uc.repo.Create(ctx, req)
}

func (uc *DatabaseUsecase) Delete(ctx context.Context, serverID uint, name string) error {
	return uc.repo.Delete(ctx, serverID, name)
}

func (uc *DatabaseUsecase) Comment(req *request.DatabaseComment) error {
	return uc.repo.Comment(req)
}
