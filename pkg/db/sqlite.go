package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// SQLite 通过 database/sql 操作 SQLite 文件
type SQLite struct {
	db   *sql.DB
	path string
}

// NewSQLite 打开 SQLite 数据库文件
func NewSQLite(ctx context.Context, path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite failed: %w", err)
	}
	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connect to sqlite failed: %w", err)
	}
	return &SQLite{db: db, path: path}, nil
}

func (r *SQLite) Close() {
	_ = r.db.Close()
}

func (r *SQLite) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// Tables 获取所有表
func (r *SQLite) Tables(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) { _ = rows.Close() }(rows)

	var tables []string
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			continue
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// SQLiteColumn 表列信息
type SQLiteColumn struct {
	CID        int    `json:"cid"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	NotNull    bool   `json:"not_null"`
	Default    string `json:"default"`
	PrimaryKey bool   `json:"primary_key"`
}

// TableInfo 获取表结构
func (r *SQLite) TableInfo(ctx context.Context, name string) ([]SQLiteColumn, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info('%s')", name))
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) { _ = rows.Close() }(rows)

	var columns []SQLiteColumn
	for rows.Next() {
		var col SQLiteColumn
		var dflt sql.NullString
		if err = rows.Scan(&col.CID, &col.Name, &col.Type, &col.NotNull, &dflt, &col.PrimaryKey); err != nil {
			continue
		}
		col.Default = dflt.String
		columns = append(columns, col)
	}
	return columns, rows.Err()
}
