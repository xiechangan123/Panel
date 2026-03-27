package db

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/acepanel/panel/v3/pkg/shell"
)

// MongoDB 通过 mongosh CLI 操作 MongoDB
type MongoDB struct {
	username string
	password string
	address  string // host:port
}

// NewMongoDB 创建 MongoDB 连接
func NewMongoDB(username, password, address string) (*MongoDB, error) {
	m := &MongoDB{
		username: username,
		password: password,
		address:  address,
	}

	if err := m.Ping(); err != nil {
		return nil, fmt.Errorf("connect to mongodb failed: %w", err)
	}

	return m, nil
}

func (r *MongoDB) Close() {}

func (r *MongoDB) Ping() error {
	_, err := r.mongosh(`db.runCommand({ping:1})`)
	return err
}

// DatabaseCreate 创建数据库（MongoDB 通过创建集合来显式创建数据库）
func (r *MongoDB) DatabaseCreate(name string) error {
	_, err := r.mongosh(fmt.Sprintf(`db.getSiblingDB('%s').createCollection('_init')`, name))
	return err
}

// DatabaseDrop 删除数据库
func (r *MongoDB) DatabaseDrop(name string) error {
	_, err := r.mongosh(fmt.Sprintf(`db.getSiblingDB('%s').dropDatabase()`, name))
	return err
}

// Databases 获取数据库列表
func (r *MongoDB) Databases() ([]MongoDatabase, error) {
	raw, err := r.mongosh(`JSON.stringify(db.adminCommand({listDatabases:1,nameOnly:false}))`)
	if err != nil {
		return nil, err
	}

	var result struct {
		Databases []struct {
			Name       string `json:"name"`
			SizeOnDisk any    `json:"sizeOnDisk"`
		} `json:"databases"`
	}
	if err = json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("failed to parse databases: %w", err)
	}

	var databases []MongoDatabase
	for _, db := range result.Databases {
		if db.Name == "admin" || db.Name == "config" || db.Name == "local" {
			continue
		}
		databases = append(databases, MongoDatabase{
			Name:       db.Name,
			SizeOnDisk: mongoLongToInt64(db.SizeOnDisk),
		})
	}

	return databases, nil
}

// mongoLongToInt64 将 MongoDB Long 对象 {"high":0,"low":8192,"unsigned":false} 转换为 int64
func mongoLongToInt64(v any) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case map[string]any:
		high, _ := val["high"].(float64)
		low, _ := val["low"].(float64)
		return int64(high)*4294967296 + int64(low)
	default:
		return 0
	}
}

// UserCreate 创建用户
func (r *MongoDB) UserCreate(user, password, database string) error {
	_, err := r.mongosh(fmt.Sprintf(`db.getSiblingDB('%s').createUser({user:'%s',pwd:'%s',roles:[{role:'readWrite',db:'%s'}]})`, database, user, password, database))
	return err
}

// UserDrop 删除用户
func (r *MongoDB) UserDrop(user, database string) error {
	_, err := r.mongosh(fmt.Sprintf(`db.getSiblingDB('%s').dropUser('%s')`, database, user))
	return err
}

// UserPassword 修改用户密码
func (r *MongoDB) UserPassword(user, password string) error {
	_, err := r.mongosh(fmt.Sprintf(`db.getSiblingDB('admin').changeUserPassword('%s','%s')`, user, password))
	return err
}

// Users 获取用户列表
func (r *MongoDB) Users() ([]MongoUser, error) {
	raw, err := r.mongosh(`JSON.stringify(db.getSiblingDB('admin').system.users.find({},{user:1,db:1,roles:1}).toArray())`)
	if err != nil {
		return nil, err
	}

	var result []struct {
		User  string `json:"user"`
		DB    string `json:"db"`
		Roles []struct {
			Role string `json:"role"`
			DB   string `json:"db"`
		} `json:"roles"`
	}
	if err = json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("failed to parse users: %w", err)
	}

	var users []MongoUser
	for _, u := range result {
		var roles []string
		for _, role := range u.Roles {
			roles = append(roles, fmt.Sprintf("%s@%s", role.Role, role.DB))
		}
		users = append(users, MongoUser{
			User:  u.User,
			DB:    u.DB,
			Roles: roles,
		})
	}

	return users, nil
}

// mongosh 执行 mongosh 命令
func (r *MongoDB) mongosh(eval string) (string, error) {
	cmd := fmt.Sprintf(`mongosh --quiet --eval "%s" mongodb://%s:%s@%s/admin 2>/dev/null`,
		strings.ReplaceAll(eval, `"`, `\"`),
		r.username, r.password, r.address,
	)
	raw, err := shell.Execf(cmd)
	if err != nil {
		return "", fmt.Errorf("mongosh error: %w", err)
	}
	return strings.TrimSpace(raw), nil
}

// MongoDatabase MongoDB 数据库信息
type MongoDatabase struct {
	Name       string `json:"name"`
	SizeOnDisk int64  `json:"size_on_disk"`
}

// MongoUser MongoDB 用户信息
type MongoUser struct {
	User  string   `json:"user"`
	DB    string   `json:"db"`
	Roles []string `json:"roles"`
}
