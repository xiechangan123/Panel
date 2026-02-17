package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

type RedisKV struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Type      string    `json:"type"`
	Size      int64     `json:"size"`
	Length    int64     `json:"length"`
	TTL       int64     `json:"ttl"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Redis struct {
	conn     redis.Conn
	username string
	password string
	address  string
}

func NewRedis(username, password, address string) (*Redis, error) {
	conn, err := redis.Dial("tcp", address, redis.DialUsername(username), redis.DialPassword(password))
	if err != nil {
		return nil, err
	}

	return &Redis{
		conn:     conn,
		username: username,
		password: password,
		address:  address,
	}, nil
}

func (r *Redis) Close() {
	_ = r.conn.Close()
}

func (r *Redis) Exec(command string, args ...any) (any, error) {
	return r.conn.Do(command, args...)
}

// Database 获取数据库数量
func (r *Redis) Database() (int, error) {
	values, err := redis.Strings(r.conn.Do("CONFIG", "GET", "databases"))
	if err != nil {
		return 0, err
	}
	if len(values) < 2 {
		return 16, nil // 默认 16 个数据库
	}
	var count int
	if _, err = fmt.Sscanf(values[1], "%d", &count); err != nil {
		return 16, nil
	}
	return count, nil
}

func (r *Redis) Select(db int) error {
	_, err := r.conn.Do("SELECT", db)
	return err
}

func (r *Redis) Size() (int, error) {
	return redis.Int(r.conn.Do("DBSIZE"))
}

func (r *Redis) Data(page, pageSize int) ([]RedisKV, int, error) {
	return r.Search("", page, pageSize)
}

// Search 搜索匹配的 key，支持分页
func (r *Redis) Search(pattern string, page, pageSize int) ([]RedisKV, int, error) {
	keys, err := r.scanKeys(pattern)
	if err != nil {
		return nil, 0, err
	}

	total := len(keys)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		return []RedisKV{}, total, nil
	}
	if end > total {
		end = total
	}

	result := make([]RedisKV, 0, end-start)
	for _, key := range keys[start:end] {
		kv := RedisKV{Key: key}
		r.fillKeyMeta(&kv)
		if kv.Type == "" || kv.Type == "none" {
			continue
		}
		if err = r.fillKeyValue(&kv); err != nil {
			continue
		}
		result = append(result, kv)
	}

	return result, total, nil
}

// Get 获取单个 key 的完整信息
func (r *Redis) Get(key string) (*RedisKV, error) {
	keyType, err := redis.String(r.conn.Do("TYPE", key))
	if err != nil {
		return nil, fmt.Errorf("key not found: %v", err)
	}
	if keyType == "none" {
		return nil, fmt.Errorf("key not found")
	}

	kv := &RedisKV{Key: key, Type: keyType}
	r.fillKeyMeta(kv)
	if err = r.fillKeyValue(kv); err != nil {
		return nil, err
	}
	return kv, nil
}

// Del 删除 key
func (r *Redis) Del(keys ...string) error {
	args := make([]any, len(keys))
	for i, key := range keys {
		args[i] = key
	}
	_, err := r.conn.Do("DEL", args...)
	return err
}

// Expire 设置 key 的 TTL，ttl 为 -1 时移除过期时间
func (r *Redis) Expire(key string, ttl int64) error {
	if ttl < 0 {
		_, err := r.conn.Do("PERSIST", key)
		return err
	}
	_, err := r.conn.Do("EXPIRE", key, ttl)
	return err
}

// Rename 重命名 key
func (r *Redis) Rename(oldKey, newKey string) error {
	_, err := r.conn.Do("RENAME", oldKey, newKey)
	return err
}

// SetKey 根据类型创建或设置 key
func (r *Redis) SetKey(key, value, keyType string, ttl int64) error {
	switch keyType {
	case "string":
		if _, err := r.conn.Do("SET", key, value); err != nil {
			return err
		}
	case "list":
		var items []string
		if err := json.Unmarshal([]byte(value), &items); err != nil {
			return fmt.Errorf("list value must be JSON array: %v", err)
		}
		_, _ = r.conn.Do("DEL", key)
		for _, item := range items {
			if _, err := r.conn.Do("RPUSH", key, item); err != nil {
				return err
			}
		}
	case "set":
		var items []string
		if err := json.Unmarshal([]byte(value), &items); err != nil {
			return fmt.Errorf("set value must be JSON array: %v", err)
		}
		_, _ = r.conn.Do("DEL", key)
		for _, item := range items {
			if _, err := r.conn.Do("SADD", key, item); err != nil {
				return err
			}
		}
	case "zset":
		var members map[string]string
		if err := json.Unmarshal([]byte(value), &members); err != nil {
			return fmt.Errorf("zset value must be JSON object {member: score}: %v", err)
		}
		_, _ = r.conn.Do("DEL", key)
		for member, score := range members {
			if _, err := r.conn.Do("ZADD", key, score, member); err != nil {
				return err
			}
		}
	case "hash":
		var fields map[string]string
		if err := json.Unmarshal([]byte(value), &fields); err != nil {
			return fmt.Errorf("hash value must be JSON object: %v", err)
		}
		_, _ = r.conn.Do("DEL", key)
		for field, val := range fields {
			if _, err := r.conn.Do("HSET", key, field, val); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported type: %s", keyType)
	}

	if ttl > 0 {
		if _, err := r.conn.Do("EXPIRE", key, ttl); err != nil {
			return err
		}
	}

	return nil
}

func (r *Redis) Clear() error {
	_, err := r.conn.Do("FLUSHDB")
	return err
}

// scanKeys 使用 SCAN 遍历所有匹配的 key
func (r *Redis) scanKeys(pattern string) ([]string, error) {
	var keys []string
	cursor := 0
	args := []any{cursor, "COUNT", 100}
	if pattern != "" && pattern != "*" {
		args = []any{cursor, "MATCH", pattern, "COUNT", 100}
	}
	for {
		args[0] = cursor
		values, err := redis.Values(r.conn.Do("SCAN", args...))
		if err != nil {
			return nil, fmt.Errorf("failed to SCAN: %v", err)
		}
		var batch []string
		if _, err = redis.Scan(values, &cursor, &batch); err != nil {
			return nil, fmt.Errorf("failed to parse SCAN result: %v", err)
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

// fillKeyValue 填充 key 的值信息
func (r *Redis) fillKeyValue(kv *RedisKV) error {
	var value any
	var err error
	switch kv.Type {
	case "string":
		if value, err = redis.String(r.conn.Do("GET", kv.Key)); err == nil {
			kv.Length = int64(len(value.(string)))
		}
	case "list":
		if value, err = redis.Strings(r.conn.Do("LRANGE", kv.Key, 0, -1)); err == nil {
			kv.Length, _ = redis.Int64(r.conn.Do("LLEN", kv.Key))
		}
	case "set":
		if value, err = redis.Strings(r.conn.Do("SMEMBERS", kv.Key)); err == nil {
			kv.Length, _ = redis.Int64(r.conn.Do("SCARD", kv.Key))
		}
	case "zset":
		if members, e := redis.Strings(r.conn.Do("ZRANGE", kv.Key, 0, -1, "WITHSCORES")); e == nil {
			kv.Length, _ = redis.Int64(r.conn.Do("ZCARD", kv.Key))
			zsetMap := make(map[string]string)
			for i := 0; i < len(members); i += 2 {
				zsetMap[members[i]] = members[i+1]
			}
			value = zsetMap
		} else {
			err = e
		}
	case "hash":
		if value, err = redis.StringMap(r.conn.Do("HGETALL", kv.Key)); err == nil {
			kv.Length, _ = redis.Int64(r.conn.Do("HLEN", kv.Key))
		}
	default:
		return fmt.Errorf("unsupported type: %s", kv.Type)
	}
	if err != nil {
		return err
	}
	if kv.Length > 5000 {
		value = "data is too long, can't display"
	}
	if str, ok := value.(string); ok {
		kv.Value = str
	} else {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		kv.Value = string(encoded)
	}
	return nil
}

// fillKeyMeta 填充 key 的元信息（类型、TTL、大小）
func (r *Redis) fillKeyMeta(kv *RedisKV) {
	if kv.Type == "" {
		kv.Type, _ = redis.String(r.conn.Do("TYPE", kv.Key))
	}
	ttl, err := redis.Int64(r.conn.Do("TTL", kv.Key))
	if err == nil {
		kv.TTL = ttl
	}
	idleTime, err := redis.Int64(r.conn.Do("OBJECT", "IDLETIME", kv.Key))
	if err == nil {
		kv.UpdatedAt = time.Now().Add(-time.Duration(idleTime) * time.Second)
	} else {
		kv.UpdatedAt = time.Now()
	}
	memory, err := redis.Int64(r.conn.Do("MEMORY", "USAGE", kv.Key))
	if err == nil {
		kv.Size = memory
	}
}
