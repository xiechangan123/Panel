package data

import (
	"fmt"
	"log/slog"

	"github.com/leonelquinteros/gotext"
	"gorm.io/gorm"

	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/internal/http/request"
	"github.com/acepanel/panel/pkg/db"
)

type databaseRedisRepo struct {
	t   *gotext.Locale
	orm *gorm.DB
	log *slog.Logger
}

func NewDatabaseRedisRepo(t *gotext.Locale, orm *gorm.DB, log *slog.Logger) biz.DatabaseRedisRepo {
	return &databaseRedisRepo{
		t:   t,
		orm: orm,
		log: log,
	}
}

func (r *databaseRedisRepo) Databases(req *request.DatabaseRedisDatabases) (int, error) {
	client, err := r.getClient(req.ServerID, 0)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	return client.Database()
}

func (r *databaseRedisRepo) Data(req *request.DatabaseRedisData) ([]db.RedisKV, int, error) {
	client, err := r.getClient(req.ServerID, req.DB)
	if err != nil {
		return nil, 0, err
	}
	defer client.Close()

	pattern := req.Search
	if pattern == "" {
		pattern = "*"
	}

	return client.Search(pattern, int(req.Page), int(req.Limit))
}

func (r *databaseRedisRepo) KeyGet(req *request.DatabaseRedisKeyGet) (*db.RedisKV, error) {
	client, err := r.getClient(req.ServerID, req.DB)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return client.Get(req.Key)
}

func (r *databaseRedisRepo) KeySet(req *request.DatabaseRedisKeySet) error {
	client, err := r.getClient(req.ServerID, req.DB)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.SetKey(req.Key, req.Value, req.Type, req.TTL)
}

func (r *databaseRedisRepo) KeyDelete(req *request.DatabaseRedisKeyDelete) error {
	client, err := r.getClient(req.ServerID, req.DB)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Del(req.Key)
}

func (r *databaseRedisRepo) KeyTTL(req *request.DatabaseRedisKeyTTL) error {
	client, err := r.getClient(req.ServerID, req.DB)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Expire(req.Key, req.TTL)
}

func (r *databaseRedisRepo) KeyRename(req *request.DatabaseRedisKeyRename) error {
	client, err := r.getClient(req.ServerID, req.DB)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Rename(req.OldKey, req.NewKey)
}

func (r *databaseRedisRepo) Clear(req *request.DatabaseRedisClear) error {
	client, err := r.getClient(req.ServerID, req.DB)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Clear()
}

// getClient 根据服务器 ID 创建 Redis 客户端并选择指定数据库
func (r *databaseRedisRepo) getClient(serverID uint, dbIndex int) (*db.Redis, error) {
	server := new(biz.DatabaseServer)
	if err := r.orm.Where("id = ?", serverID).First(server).Error; err != nil {
		return nil, fmt.Errorf(r.t.Get("server not found"))
	}
	if server.Type != biz.DatabaseTypeRedis {
		return nil, fmt.Errorf(r.t.Get("server is not Redis type"))
	}

	client, err := db.NewRedis(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
	if err != nil {
		return nil, fmt.Errorf(r.t.Get("failed to connect to Redis: %v"), err)
	}

	if dbIndex > 0 {
		if err = client.Select(dbIndex); err != nil {
			client.Close()
			return nil, fmt.Errorf(r.t.Get("failed to select database: %v"), err)
		}
	}

	return client, nil
}
