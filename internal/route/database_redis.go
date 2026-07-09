package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/db"
)

// DatabaseRedisRoutes Redis 路由
func DatabaseRedisRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.DatabaseRedisService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/database_redis/databases", Handler: svc.Databases,
			Summary: "获取数据库数量", Tags: []string{"Redis"},
			Request: request.DatabaseRedisDatabases{}},
		{Method: http.MethodGet, Path: "/api/database_redis/data", Handler: svc.Data,
			Summary: "获取键值列表", Tags: []string{"Redis"},
			Request: request.DatabaseRedisData{}, Response: service.Envelope[service.Page[db.RedisKV]]{}},
		{Method: http.MethodGet, Path: "/api/database_redis/key", Handler: svc.KeyGet,
			Summary: "获取键值", Tags: []string{"Redis"},
			Request: request.DatabaseRedisKeyGet{}, Response: service.Envelope[db.RedisKV]{}},
		{Method: http.MethodPost, Path: "/api/database_redis/key", Handler: svc.KeySet,
			Summary: "设置键值", Tags: []string{"Redis"},
			Request: request.DatabaseRedisKeySet{}},
		{Method: http.MethodDelete, Path: "/api/database_redis/key", Handler: svc.KeyDelete,
			Summary: "删除键值", Tags: []string{"Redis"},
			Request: request.DatabaseRedisKeyDelete{}},
		{Method: http.MethodPost, Path: "/api/database_redis/key/ttl", Handler: svc.KeyTTL,
			Summary: "设置键值过期时间", Tags: []string{"Redis"},
			Request: request.DatabaseRedisKeyTTL{}},
		{Method: http.MethodPost, Path: "/api/database_redis/key/rename", Handler: svc.KeyRename,
			Summary: "重命名键值", Tags: []string{"Redis"},
			Request: request.DatabaseRedisKeyRename{}},
		{Method: http.MethodPost, Path: "/api/database_redis/clear", Handler: svc.Clear,
			Summary: "清空数据库", Tags: []string{"Redis"},
			Request: request.DatabaseRedisClear{}},
	}, nil
}
