package data

import (
	"fmt"
	"slices"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/pkg/db"
)

type databaseRepo struct {
	db *gorm.DB
}

func NewDatabaseRepo(i do.Injector) (biz.DatabaseRepo, error) {
	return &databaseRepo{
		db: do.MustInvoke[*gorm.DB](i),
	}, nil
}

// ListServers 列出数据库服务器
func (r *databaseRepo) ListServers(typ string) ([]*biz.DatabaseServer, error) {
	var databaseServer []*biz.DatabaseServer
	query := r.db.Model(&biz.DatabaseServer{}).Order("id desc")
	if typ != "" {
		query = query.Where("type = ?", typ)
	}
	if err := query.Find(&databaseServer).Error; err != nil {
		return nil, err
	}
	return databaseServer, nil
}

// DatabasesOf 列出单个服务器上的数据库
func (r *databaseRepo) DatabasesOf(server *biz.DatabaseServer) ([]*biz.Database, error) {
	database := make([]*biz.Database, 0)
	switch server.Type {
	case biz.DatabaseTypeMongoDB:
		mongo, err := db.NewMongoDB(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
		if err != nil {
			return nil, err
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
			return nil, err
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
		operator, err := r.Operator(server)
		if err != nil {
			return nil, err
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
	return database, nil
}

// Mongo 构建 MongoDB 客户端
func (r *databaseRepo) Mongo(server *biz.DatabaseServer) (*db.MongoDB, error) {
	return db.NewMongoDB(server.Username, server.Password, fmt.Sprintf("%s:%d", server.Host, server.Port))
}

func (r *databaseRepo) Operator(server *biz.DatabaseServer) (db.Operator, error) {
	switch server.Type {
	case biz.DatabaseTypeMysql:
		return newMySQLOperator(server.Username, server.Password, server.Host, server.Port)
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
