package data

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/leonelquinteros/gotext"
	"gorm.io/gorm"

	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/internal/http/request"
	"github.com/acepanel/panel/v3/pkg/db"
)

type databaseElasticsearchRepo struct {
	t   *gotext.Locale
	orm *gorm.DB
	log *slog.Logger
}

func NewDatabaseElasticsearchRepo(t *gotext.Locale, orm *gorm.DB, log *slog.Logger) biz.DatabaseElasticsearchRepo {
	return &databaseElasticsearchRepo{
		t:   t,
		orm: orm,
		log: log,
	}
}

func (r *databaseElasticsearchRepo) Indices(req *request.DatabaseESIndices) ([]db.ESIndex, error) {
	client, err := r.getClient(req.ServerID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return client.Indices()
}

func (r *databaseElasticsearchRepo) IndexCreate(req *request.DatabaseESIndexCreate) error {
	client, err := r.getClient(req.ServerID)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.IndexCreate(req.Name)
}

func (r *databaseElasticsearchRepo) IndexDelete(req *request.DatabaseESIndexDelete) error {
	client, err := r.getClient(req.ServerID)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.IndexDelete(req.Name)
}

func (r *databaseElasticsearchRepo) Data(req *request.DatabaseESData) ([]db.ESDocument, int64, error) {
	client, err := r.getClient(req.ServerID)
	if err != nil {
		return nil, 0, err
	}
	defer client.Close()

	return client.Search(req.Index, req.Search, int(req.Page), int(req.Limit))
}

func (r *databaseElasticsearchRepo) DocumentGet(req *request.DatabaseESDocumentGet) (*db.ESDocument, error) {
	client, err := r.getClient(req.ServerID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return client.DocumentGet(req.Index, req.ID)
}

func (r *databaseElasticsearchRepo) DocumentSet(req *request.DatabaseESDocumentSet) error {
	client, err := r.getClient(req.ServerID)
	if err != nil {
		return err
	}
	defer client.Close()

	if req.ID == "" {
		return client.DocumentCreate(req.Index, req.Body)
	}
	return client.DocumentUpdate(req.Index, req.ID, req.Body)
}

func (r *databaseElasticsearchRepo) DocumentDelete(req *request.DatabaseESDocumentDelete) error {
	client, err := r.getClient(req.ServerID)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.DocumentDelete(req.Index, req.ID)
}

// getClient 根据服务器 ID 创建 Elasticsearch 客户端
func (r *databaseElasticsearchRepo) getClient(serverID uint) (*db.Elasticsearch, error) {
	server := new(biz.DatabaseServer)
	if err := r.orm.Where("id = ?", serverID).First(server).Error; err != nil {
		return nil, errors.New(r.t.Get("server not found"))
	}
	if server.Type != biz.DatabaseTypeElasticsearch {
		return nil, errors.New(r.t.Get("server is not Elasticsearch type"))
	}

	client, err := db.NewElasticsearch(fmt.Sprintf("%s:%d", server.Host, server.Port), server.Username, server.Password)
	if err != nil {
		return nil, errors.New(r.t.Get("failed to connect to Elasticsearch: %v", err))
	}

	return client, nil
}
