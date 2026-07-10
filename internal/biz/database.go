package biz

import (
	"context"
	"errors"
	"log/slog"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/db"
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
	ListServers(typ string) ([]*DatabaseServer, error)
	DatabasesOf(server *DatabaseServer) ([]*Database, error)
	Operator(server *DatabaseServer) (db.Operator, error)
	Mongo(server *DatabaseServer) (*db.MongoDB, error)
}

// DatabaseUsecase 数据库业务用例
type DatabaseUsecase struct {
	repo   DatabaseRepo
	server DatabaseServerRepo
	user   *DatabaseUserUsecase
	t      *gotext.Locale
	log    *slog.Logger
}

func NewDatabaseUsecase(i do.Injector) (*DatabaseUsecase, error) {
	return &DatabaseUsecase{
		repo:   do.MustInvoke[DatabaseRepo](i),
		server: do.MustInvoke[DatabaseServerRepo](i),
		user:   do.MustInvoke[*DatabaseUserUsecase](i),
		t:      do.MustInvoke[*gotext.Locale](i),
		log:    do.MustInvoke[*slog.Logger](i),
	}, nil
}

func (uc *DatabaseUsecase) List(page, limit uint, typ string) ([]*Database, int64, error) {
	servers, err := uc.repo.ListServers(typ)
	if err != nil {
		return nil, 0, err
	}

	database := make([]*Database, 0)
	for _, server := range servers {
		databases, err := uc.repo.DatabasesOf(server)
		if err != nil {
			continue
		}
		database = append(database, databases...)
	}

	if len(database) < int((page-1)*limit) {
		return []*Database{}, int64(len(database)), nil
	}

	return database[(page-1)*limit:], int64(len(database)), nil
}

func (uc *DatabaseUsecase) Create(ctx context.Context, req *request.DatabaseCreate) error {
	server, err := uc.server.Get(req.ServerID)
	if err != nil {
		return err
	}

	// MongoDB 独立处理，不走 Operator 接口
	if server.Type == DatabaseTypeMongoDB {
		mongo, mongoErr := uc.repo.Mongo(server)
		if mongoErr != nil {
			return mongoErr
		}
		defer mongo.Close()
		if mongoErr = mongo.DatabaseCreate(req.Name); mongoErr != nil {
			return mongoErr
		}
		uc.log.Info("database created", slog.String("type", OperationTypeDatabase), slog.Uint64("operator_id", operatorID(ctx)), slog.String("name", req.Name), slog.Uint64("server_id", uint64(req.ServerID)))
		return nil
	}

	operator, err := uc.repo.Operator(server)
	if err != nil {
		return err
	}
	defer operator.Close()

	switch server.Type {
	case DatabaseTypeMysql:
		if req.CreateUser {
			if err = uc.user.Create(ctx, &request.DatabaseUserCreate{
				ServerID: req.ServerID,
				Username: req.Username,
				Password: req.Password,
				Host:     req.Host,
			}); err != nil {
				return err
			}
		}
		if err = operator.DatabaseCreate(req.Name); err != nil {
			return err
		}
		if req.Username != "" {
			if err = operator.PrivilegesGrant(req.Username, req.Name, req.Host); err != nil {
				return err
			}
		}
	case DatabaseTypePostgresql:
		if req.CreateUser {
			if err = uc.user.Create(ctx, &request.DatabaseUserCreate{
				ServerID: req.ServerID,
				Username: req.Username,
				Password: req.Password,
				Host:     req.Host,
			}); err != nil {
				return err
			}
		}
		if err = operator.DatabaseCreate(req.Name); err != nil {
			return err
		}
		if req.Username != "" {
			if err = operator.PrivilegesGrant(req.Username, req.Name); err != nil {
				return err
			}
		}
		if err = operator.(*db.Postgres).DatabaseComment(req.Name, req.Comment); err != nil {
			return err
		}
	case DatabaseTypeClickHouse:
		if req.CreateUser {
			if err = uc.user.Create(ctx, &request.DatabaseUserCreate{
				ServerID: req.ServerID,
				Username: req.Username,
				Password: req.Password,
			}); err != nil {
				return err
			}
		}
		if err = operator.DatabaseCreate(req.Name); err != nil {
			return err
		}
		if req.Username != "" {
			if err = operator.PrivilegesGrant(req.Username, req.Name); err != nil {
				return err
			}
		}
	}

	// 记录日志
	uc.log.Info("database created", slog.String("type", OperationTypeDatabase), slog.Uint64("operator_id", operatorID(ctx)), slog.String("name", req.Name), slog.Uint64("server_id", uint64(req.ServerID)))

	return nil
}

func (uc *DatabaseUsecase) Delete(ctx context.Context, serverID uint, name string) error {
	server, err := uc.server.Get(serverID)
	if err != nil {
		return err
	}

	switch server.Type {
	case DatabaseTypeMongoDB:
		mongo, mongoErr := uc.repo.Mongo(server)
		if mongoErr != nil {
			return mongoErr
		}
		defer mongo.Close()
		if mongoErr = mongo.DatabaseDrop(name); mongoErr != nil {
			return mongoErr
		}
	case DatabaseTypeSQLite:
		return errors.New(uc.t.Get("sqlite does not support dropping tables from here"))
	default:
		operator, opErr := uc.repo.Operator(server)
		if opErr != nil {
			return opErr
		}
		defer operator.Close()
		if opErr = operator.DatabaseDrop(name); opErr != nil {
			return opErr
		}
	}

	// 记录日志
	uc.log.Info("database deleted", slog.String("type", OperationTypeDatabase), slog.Uint64("operator_id", operatorID(ctx)), slog.String("name", name), slog.Uint64("server_id", uint64(serverID)))

	return nil
}

func (uc *DatabaseUsecase) Comment(req *request.DatabaseComment) error {
	server, err := uc.server.Get(req.ServerID)
	if err != nil {
		return err
	}

	switch server.Type {
	case DatabaseTypePostgresql:
		operator, opErr := uc.repo.Operator(server)
		if opErr != nil {
			return opErr
		}
		defer operator.Close()
		return operator.(*db.Postgres).DatabaseComment(req.Name, req.Comment)
	default:
		return errors.New(uc.t.Get("%s does not support database comment", server.Type))
	}
}
