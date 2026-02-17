package request

type DatabaseRedisDatabases struct {
	ServerID uint `form:"server_id" json:"server_id" query:"server_id" validate:"required|exists:database_servers,id"`
}

type DatabaseRedisData struct {
	Paginate
	ServerID uint   `form:"server_id" json:"server_id" query:"server_id" validate:"required|exists:database_servers,id"`
	DB       int    `form:"db" json:"db" query:"db"`
	Search   string `form:"search" json:"search" query:"search"`
}

type DatabaseRedisKeyGet struct {
	ServerID uint   `form:"server_id" json:"server_id" query:"server_id" validate:"required|exists:database_servers,id"`
	DB       int    `form:"db" json:"db" query:"db"`
	Key      string `form:"key" json:"key" query:"key" validate:"required"`
}

type DatabaseRedisKeySet struct {
	ServerID uint   `form:"server_id" json:"server_id" validate:"required|exists:database_servers,id"`
	DB       int    `form:"db" json:"db"`
	Key      string `form:"key" json:"key" validate:"required"`
	Value    string `form:"value" json:"value" validate:"required"`
	Type     string `form:"type" json:"type" validate:"required|in:string,list,set,zset,hash"`
	TTL      int64  `form:"ttl" json:"ttl"`
}

type DatabaseRedisKeyDelete struct {
	ServerID uint   `form:"server_id" json:"server_id" validate:"required|exists:database_servers,id"`
	DB       int    `form:"db" json:"db"`
	Key      string `form:"key" json:"key" validate:"required"`
}

type DatabaseRedisKeyTTL struct {
	ServerID uint   `form:"server_id" json:"server_id" validate:"required|exists:database_servers,id"`
	DB       int    `form:"db" json:"db"`
	Key      string `form:"key" json:"key" validate:"required"`
	TTL      int64  `form:"ttl" json:"ttl"`
}

type DatabaseRedisKeyRename struct {
	ServerID uint   `form:"server_id" json:"server_id" validate:"required|exists:database_servers,id"`
	DB       int    `form:"db" json:"db"`
	OldKey   string `form:"old_key" json:"old_key" validate:"required"`
	NewKey   string `form:"new_key" json:"new_key" validate:"required"`
}

type DatabaseRedisClear struct {
	ServerID uint `form:"server_id" json:"server_id" validate:"required|exists:database_servers,id"`
	DB       int  `form:"db" json:"db"`
}
