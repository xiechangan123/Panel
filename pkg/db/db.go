package db

import (
	"context"
	"database/sql"
)

type Operator interface {
	Close()
	Ping(ctx context.Context) error

	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	Prepare(ctx context.Context, query string) (*sql.Stmt, error)

	DatabaseCreate(ctx context.Context, name string) error
	DatabaseDrop(ctx context.Context, name string) error
	DatabaseExists(ctx context.Context, name string) (bool, error)
	DatabaseSize(ctx context.Context, name string) (int64, error)

	UserCreate(ctx context.Context, user, password string, host ...string) error
	UserDrop(ctx context.Context, user string, host ...string) error
	UserPassword(ctx context.Context, user, password string, host ...string) error
	UserPrivileges(ctx context.Context, user string, host ...string) ([]string, error)

	PrivilegesGrant(ctx context.Context, user, database string, host ...string) error
	PrivilegesRevoke(ctx context.Context, user, database string, host ...string) error

	Users(ctx context.Context) ([]User, error)
	Databases(ctx context.Context) ([]Database, error)
}
