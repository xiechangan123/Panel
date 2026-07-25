package biz

import (
	"context"
	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/pkg/db"
)

type DatabaseElasticsearchRepo interface {
	Indices(ctx context.Context, req *request.DatabaseESIndices) ([]db.ESIndex, error)
	IndexCreate(ctx context.Context, req *request.DatabaseESIndexCreate) error
	IndexDelete(ctx context.Context, req *request.DatabaseESIndexDelete) error
	Data(ctx context.Context, req *request.DatabaseESData) ([]db.ESDocument, int64, error)
	DocumentGet(ctx context.Context, req *request.DatabaseESDocumentGet) (*db.ESDocument, error)
	DocumentSet(ctx context.Context, req *request.DatabaseESDocumentSet) error
	DocumentDelete(ctx context.Context, req *request.DatabaseESDocumentDelete) error
}

// DatabaseElasticsearchUsecase Elasticsearch 业务用例
type DatabaseElasticsearchUsecase struct {
	repo DatabaseElasticsearchRepo
}

func NewDatabaseElasticsearchUsecase(repo DatabaseElasticsearchRepo) *DatabaseElasticsearchUsecase {
	return &DatabaseElasticsearchUsecase{repo: repo}
}

func (uc *DatabaseElasticsearchUsecase) Indices(ctx context.Context, req *request.DatabaseESIndices) ([]db.ESIndex, error) {
	return uc.repo.Indices(ctx, req)
}

func (uc *DatabaseElasticsearchUsecase) IndexCreate(ctx context.Context, req *request.DatabaseESIndexCreate) error {
	return uc.repo.IndexCreate(ctx, req)
}

func (uc *DatabaseElasticsearchUsecase) IndexDelete(ctx context.Context, req *request.DatabaseESIndexDelete) error {
	return uc.repo.IndexDelete(ctx, req)
}

func (uc *DatabaseElasticsearchUsecase) Data(ctx context.Context, req *request.DatabaseESData) ([]db.ESDocument, int64, error) {
	return uc.repo.Data(ctx, req)
}

func (uc *DatabaseElasticsearchUsecase) DocumentGet(ctx context.Context, req *request.DatabaseESDocumentGet) (*db.ESDocument, error) {
	return uc.repo.DocumentGet(ctx, req)
}

func (uc *DatabaseElasticsearchUsecase) DocumentSet(ctx context.Context, req *request.DatabaseESDocumentSet) error {
	return uc.repo.DocumentSet(ctx, req)
}

func (uc *DatabaseElasticsearchUsecase) DocumentDelete(ctx context.Context, req *request.DatabaseESDocumentDelete) error {
	return uc.repo.DocumentDelete(ctx, req)
}
