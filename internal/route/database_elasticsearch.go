package route

import (
	"net/http"

	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/request"
	"github.com/acepanel/panel/v3/internal/service"
	"github.com/acepanel/panel/v3/pkg/db"
)

// DatabaseElasticsearchRoutes Elasticsearch 路由
func DatabaseElasticsearchRoutes(i do.Injector) (Endpoints, error) {
	svc := do.MustInvoke[*service.DatabaseElasticsearchService](i)

	return Endpoints{
		{Method: http.MethodGet, Path: "/api/database_elasticsearch/indices", Handler: svc.Indices,
			Summary: "获取索引列表", Tags: []string{"Elasticsearch"},
			Request: request.DatabaseESIndices{}, Response: service.Envelope[[]db.ESIndex]{}},
		{Method: http.MethodPost, Path: "/api/database_elasticsearch/index", Handler: svc.IndexCreate,
			Summary: "创建索引", Tags: []string{"Elasticsearch"},
			Request: request.DatabaseESIndexCreate{}},
		{Method: http.MethodDelete, Path: "/api/database_elasticsearch/index", Handler: svc.IndexDelete,
			Summary: "删除索引", Tags: []string{"Elasticsearch"},
			Request: request.DatabaseESIndexDelete{}},
		{Method: http.MethodGet, Path: "/api/database_elasticsearch/data", Handler: svc.Data,
			Summary: "获取文档列表", Tags: []string{"Elasticsearch"},
			Request: request.DatabaseESData{}, Response: service.Envelope[service.Page[db.ESDocument]]{}},
		{Method: http.MethodGet, Path: "/api/database_elasticsearch/document", Handler: svc.DocumentGet,
			Summary: "获取文档", Tags: []string{"Elasticsearch"},
			Request: request.DatabaseESDocumentGet{}, Response: service.Envelope[db.ESDocument]{}},
		{Method: http.MethodPost, Path: "/api/database_elasticsearch/document", Handler: svc.DocumentSet,
			Summary: "设置文档", Tags: []string{"Elasticsearch"},
			Request: request.DatabaseESDocumentSet{}},
		{Method: http.MethodDelete, Path: "/api/database_elasticsearch/document", Handler: svc.DocumentDelete,
			Summary: "删除文档", Tags: []string{"Elasticsearch"},
			Request: request.DatabaseESDocumentDelete{}},
	}, nil
}
