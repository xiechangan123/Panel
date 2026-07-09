package biz

import (
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/db"
)

type DatabaseElasticsearchRepo interface {
	Indices(req *request.DatabaseESIndices) ([]db.ESIndex, error)
	IndexCreate(req *request.DatabaseESIndexCreate) error
	IndexDelete(req *request.DatabaseESIndexDelete) error
	Data(req *request.DatabaseESData) ([]db.ESDocument, int64, error)
	DocumentGet(req *request.DatabaseESDocumentGet) (*db.ESDocument, error)
	DocumentSet(req *request.DatabaseESDocumentSet) error
	DocumentDelete(req *request.DatabaseESDocumentDelete) error
}

// DatabaseElasticsearchUsecase Elasticsearch 业务用例
type DatabaseElasticsearchUsecase struct {
	repo DatabaseElasticsearchRepo
}

func NewDatabaseElasticsearchUsecase(repo DatabaseElasticsearchRepo) *DatabaseElasticsearchUsecase {
	return &DatabaseElasticsearchUsecase{repo: repo}
}

func (uc *DatabaseElasticsearchUsecase) Indices(req *request.DatabaseESIndices) ([]db.ESIndex, error) {
	return uc.repo.Indices(req)
}

func (uc *DatabaseElasticsearchUsecase) IndexCreate(req *request.DatabaseESIndexCreate) error {
	return uc.repo.IndexCreate(req)
}

func (uc *DatabaseElasticsearchUsecase) IndexDelete(req *request.DatabaseESIndexDelete) error {
	return uc.repo.IndexDelete(req)
}

func (uc *DatabaseElasticsearchUsecase) Data(req *request.DatabaseESData) ([]db.ESDocument, int64, error) {
	return uc.repo.Data(req)
}

func (uc *DatabaseElasticsearchUsecase) DocumentGet(req *request.DatabaseESDocumentGet) (*db.ESDocument, error) {
	return uc.repo.DocumentGet(req)
}

func (uc *DatabaseElasticsearchUsecase) DocumentSet(req *request.DatabaseESDocumentSet) error {
	return uc.repo.DocumentSet(req)
}

func (uc *DatabaseElasticsearchUsecase) DocumentDelete(req *request.DatabaseESDocumentDelete) error {
	return uc.repo.DocumentDelete(req)
}
