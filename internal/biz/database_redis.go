package biz

import (
	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/db"
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
