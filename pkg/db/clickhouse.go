package db

import (
	"database/sql"
	"fmt"
	"strings"

	"resty.dev/v3"
)

// ClickHouse 通过 HTTP API 操作 ClickHouse
type ClickHouse struct {
	client   *resty.Client
	address  string
	username string
	password string
}

// NewClickHouse 创建 ClickHouse 连接（HTTP API）
func NewClickHouse(username, password, address string) (*ClickHouse, error) {
	client := resty.New()
	client.SetBaseURL(fmt.Sprintf("http://%s", address))
	client.SetTimeout(10 * 1000 * 1000 * 1000) // 10s

	ch := &ClickHouse{
		client:   client,
		address:  address,
		username: username,
		password: password,
	}

	// 测试连接
	if err := ch.Ping(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect to clickhouse failed: %w", err)
	}

	return ch, nil
}

func (r *ClickHouse) Close() {
	_ = r.client.Close()
}

func (r *ClickHouse) Ping() error {
	_, err := r.exec("SELECT 1")
	return err
}

func (r *ClickHouse) Query(query string, args ...any) (*sql.Rows, error) {
	return nil, fmt.Errorf("clickhouse HTTP API does not support sql.Rows")
}

func (r *ClickHouse) QueryRow(query string, args ...any) *sql.Row {
	return nil
}

func (r *ClickHouse) Exec(query string, args ...any) (sql.Result, error) {
	_, err := r.exec(query)
	return nil, err
}

func (r *ClickHouse) Prepare(query string) (*sql.Stmt, error) {
	return nil, fmt.Errorf("clickhouse HTTP API does not support Prepare")
}

func (r *ClickHouse) DatabaseCreate(name string) error {
	_, err := r.exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", name))
	return err
}

func (r *ClickHouse) DatabaseDrop(name string) error {
	_, err := r.exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name))
	return err
}

func (r *ClickHouse) DatabaseExists(name string) (bool, error) {
	result, err := r.exec(fmt.Sprintf("SELECT count() FROM system.databases WHERE name = '%s'", name))
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(result) != "0", nil
}

func (r *ClickHouse) DatabaseSize(name string) (int64, error) {
	result, err := r.exec(fmt.Sprintf("SELECT COALESCE(sum(bytes_on_disk), 0) FROM system.parts WHERE database = '%s'", name))
	if err != nil {
		return 0, err
	}
	var size int64
	if _, err = fmt.Sscanf(strings.TrimSpace(result), "%d", &size); err != nil {
		return 0, nil
	}
	return size, nil
}

func (r *ClickHouse) UserCreate(user, password string, host ...string) error {
	_, err := r.exec(fmt.Sprintf("CREATE USER IF NOT EXISTS `%s` IDENTIFIED BY '%s'", user, password))
	return err
}

func (r *ClickHouse) UserDrop(user string, host ...string) error {
	_, err := r.exec(fmt.Sprintf("DROP USER IF EXISTS `%s`", user))
	return err
}

func (r *ClickHouse) UserPassword(user, password string, host ...string) error {
	_, err := r.exec(fmt.Sprintf("ALTER USER `%s` IDENTIFIED BY '%s'", user, password))
	return err
}

func (r *ClickHouse) UserPrivileges(user string, host ...string) ([]string, error) {
	result, err := r.exec(fmt.Sprintf("SHOW GRANTS FOR `%s`", user))
	if err != nil {
		return nil, err
	}

	var databases []string
	for line := range strings.SplitSeq(result, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 解析 GRANT ALL ON database.* TO user 格式
		if idx := strings.Index(line, " ON "); idx != -1 {
			rest := line[idx+4:]
			if dotIdx := strings.Index(rest, "."); dotIdx != -1 {
				dbName := strings.Trim(rest[:dotIdx], "`")
				if dbName != "" && dbName != "*" {
					databases = append(databases, dbName)
				}
			}
		}
	}

	return databases, nil
}

func (r *ClickHouse) PrivilegesGrant(user, database string, host ...string) error {
	_, err := r.exec(fmt.Sprintf("GRANT ALL ON `%s`.* TO `%s`", database, user))
	return err
}

func (r *ClickHouse) PrivilegesRevoke(user, database string, host ...string) error {
	_, err := r.exec(fmt.Sprintf("REVOKE ALL ON `%s`.* FROM `%s`", database, user))
	return err
}

func (r *ClickHouse) Users() ([]User, error) {
	result, err := r.exec("SELECT name FROM system.users FORMAT TabSeparated")
	if err != nil {
		return nil, err
	}

	var users []User
	for line := range strings.SplitSeq(result, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		users = append(users, User{User: line})
	}

	return users, nil
}

func (r *ClickHouse) Databases() ([]Database, error) {
	result, err := r.exec("SELECT name FROM system.databases WHERE name NOT IN ('system', 'INFORMATION_SCHEMA', 'information_schema') FORMAT TabSeparated")
	if err != nil {
		return nil, err
	}

	var databases []Database
	for line := range strings.SplitSeq(result, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		databases = append(databases, Database{Name: line})
	}

	return databases, nil
}

// exec 通过 HTTP API 执行 SQL
func (r *ClickHouse) exec(query string) (string, error) {
	resp, err := r.client.R().
		SetQueryParam("query", query).
		SetQueryParam("user", r.username).
		SetQueryParam("password", r.password).
		Get("/")
	if err != nil {
		return "", fmt.Errorf("clickhouse query failed: %w", err)
	}
	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("clickhouse query error: %s", strings.TrimSpace(resp.String()))
	}
	return resp.String(), nil
}
