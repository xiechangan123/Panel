package data

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/leonelquinteros/gotext"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/http/request"
	"github.com/acepanel/panel/v3/pkg/db"
)

type databaseRepo struct {
	t      *gotext.Locale
	db     *gorm.DB
	log    *slog.Logger
	server biz.DatabaseServerRepo
	user   biz.DatabaseUserRepo
}

func NewDatabaseRepo(t *gotext.Locale, db *gorm.DB, log *slog.Logger, server biz.DatabaseServerRepo, user biz.DatabaseUserRepo) biz.DatabaseRepo {
	return &databaseRepo{
		t:      t,
		db:     db,
		log:    log,
		server: server,
		user:   user,
	}
}

func (r *databaseRepo) List(page, limit uint, typ string) ([]*biz.Database, int64, error) {
	var databaseServer []*biz.DatabaseServer
	query := r.db.Model(&biz.DatabaseServer{}).Order("id desc")
	if typ != "" {
		query = query.Where("type = ?", typ)
	}
	if err := query.Find(&databaseServer).Error; err != nil {
		return nil, 0, err
	}

	database := make([]*biz.Database, 0)
	for _, server := range databaseServer {
		switch server.Type {
		case biz.DatabaseTypeMongoDB:
			mongo, err := db.NewMongoDB(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
			if err != nil {
				continue
			}
			if databases, err := mongo.Databases(); err == nil {
				for item := range slices.Values(databases) {
					database = append(database, &biz.Database{
						Type:     server.Type,
						Name:     item.Name,
						Server:   server.Name,
						ServerID: server.ID,
					})
				}
			}
			mongo.Close()
		case biz.DatabaseTypeSQLite:
			sqlite, err := db.NewSQLite(server.Host)
			if err != nil {
				continue
			}
			if tables, err := sqlite.Tables(); err == nil {
				for table := range slices.Values(tables) {
					database = append(database, &biz.Database{
						Type:     server.Type,
						Name:     table,
						Server:   server.Name,
						ServerID: server.ID,
					})
				}
			}
			sqlite.Close()
		default:
			operator, err := r.getOperator(server)
			if err != nil {
				continue
			}
			if databases, err := operator.Databases(); err == nil {
				for item := range slices.Values(databases) {
					database = append(database, &biz.Database{
						Type:     server.Type,
						Name:     item.Name,
						Server:   server.Name,
						ServerID: server.ID,
						Encoding: item.CharSet,
						Comment:  item.Comment,
					})
				}
			}
			operator.Close()
		}
	}

	if len(database) < int((page-1)*limit) {
		return []*biz.Database{}, int64(len(database)), nil
	}

	return database[(page-1)*limit:], int64(len(database)), nil
}

func (r *databaseRepo) Create(ctx context.Context, req *request.DatabaseCreate) error {
	server, err := r.server.Get(req.ServerID)
	if err != nil {
		return err
	}

	operator, err := r.getOperator(server)
	if err != nil {
		return err
	}
	defer operator.Close()

	switch server.Type {
	case biz.DatabaseTypeMysql:
		if req.CreateUser {
			if err = r.user.Create(ctx, &request.DatabaseUserCreate{
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
	case biz.DatabaseTypePostgresql:
		if req.CreateUser {
			if err = r.user.Create(ctx, &request.DatabaseUserCreate{
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
	case biz.DatabaseTypeClickHouse:
		if req.CreateUser {
			if err = r.user.Create(ctx, &request.DatabaseUserCreate{
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
	case biz.DatabaseTypeMongoDB:
		mongo, mongoErr := db.NewMongoDB(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
		if mongoErr != nil {
			return mongoErr
		}
		defer mongo.Close()
		if mongoErr = mongo.DatabaseCreate(req.Name); mongoErr != nil {
			return mongoErr
		}
	}

	// 记录日志
	r.log.Info("database created", slog.String("type", biz.OperationTypeDatabase), slog.Uint64("operator_id", getOperatorID(ctx)), slog.String("name", req.Name), slog.Uint64("server_id", uint64(req.ServerID)))

	return nil
}

func (r *databaseRepo) Delete(ctx context.Context, serverID uint, name string) error {
	server, err := r.server.Get(serverID)
	if err != nil {
		return err
	}

	switch server.Type {
	case biz.DatabaseTypeMongoDB:
		mongo, mongoErr := db.NewMongoDB(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
		if mongoErr != nil {
			return mongoErr
		}
		defer mongo.Close()
		if mongoErr = mongo.DatabaseDrop(name); mongoErr != nil {
			return mongoErr
		}
	case biz.DatabaseTypeSQLite:
		return errors.New(r.t.Get("sqlite does not support dropping tables from here"))
	default:
		operator, opErr := r.getOperator(server)
		if opErr != nil {
			return opErr
		}
		defer operator.Close()
		if opErr = operator.DatabaseDrop(name); opErr != nil {
			return opErr
		}
	}

	// 记录日志
	r.log.Info("database deleted", slog.String("type", biz.OperationTypeDatabase), slog.Uint64("operator_id", getOperatorID(ctx)), slog.String("name", name), slog.Uint64("server_id", uint64(serverID)))

	return nil
}

func (r *databaseRepo) Comment(req *request.DatabaseComment) error {
	server, err := r.server.Get(req.ServerID)
	if err != nil {
		return err
	}

	operator, err := r.getOperator(server)
	if err != nil {
		return err
	}
	defer operator.Close()

	switch server.Type {
	case biz.DatabaseTypeMysql:
		return errors.New(r.t.Get("mysql not support database comment"))
	case biz.DatabaseTypePostgresql:
		return operator.(*db.Postgres).DatabaseComment(req.Name, req.Comment)
	}

	return nil
}

func (r *databaseRepo) getOperator(server *biz.DatabaseServer) (db.Operator, error) {
	switch server.Type {
	case biz.DatabaseTypeMysql:
		mysql, err := db.NewMySQL(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
		if err != nil {
			return nil, err
		}
		return mysql, nil
	case biz.DatabaseTypePostgresql:
		postgres, err := db.NewPostgres(server.Username, server.Password, server.Host, server.Port)
		if err != nil {
			return nil, err
		}
		return postgres, nil
	case biz.DatabaseTypeClickHouse:
		clickhouse, err := db.NewClickHouse(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
		if err != nil {
			return nil, err
		}
		return clickhouse, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", server.Type)
	}
}
