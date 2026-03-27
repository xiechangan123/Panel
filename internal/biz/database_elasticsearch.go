package biz

import (
	"github.com/acepanel/panel/v3/internal/http/request"
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
