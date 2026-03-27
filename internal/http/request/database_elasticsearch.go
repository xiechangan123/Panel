package request

// DatabaseESIndices 获取索引列表
type DatabaseESIndices struct {
	ServerID uint `form:"server_id" json:"server_id" query:"server_id" validate:"required|exists:database_servers,id"`
}

// DatabaseESIndexCreate 创建索引
type DatabaseESIndexCreate struct {
	ServerID uint   `form:"server_id" json:"server_id" validate:"required|exists:database_servers,id"`
	Name     string `form:"name" json:"name" validate:"required"`
}

// DatabaseESIndexDelete 删除索引
type DatabaseESIndexDelete struct {
	ServerID uint   `form:"server_id" json:"server_id" validate:"required|exists:database_servers,id"`
	Name     string `form:"name" json:"name" validate:"required"`
}

// DatabaseESData 搜索文档列表
type DatabaseESData struct {
	Paginate
	ServerID uint   `form:"server_id" json:"server_id" query:"server_id" validate:"required|exists:database_servers,id"`
	Index    string `form:"index" json:"index" query:"index" validate:"required"`
	Search   string `form:"search" json:"search" query:"search"`
}

// DatabaseESDocumentGet 获取文档
type DatabaseESDocumentGet struct {
	ServerID uint   `form:"server_id" json:"server_id" query:"server_id" validate:"required|exists:database_servers,id"`
	Index    string `form:"index" json:"index" query:"index" validate:"required"`
	ID       string `form:"id" json:"id" query:"id" validate:"required"`
}

// DatabaseESDocumentSet 创建/更新文档
type DatabaseESDocumentSet struct {
	ServerID uint   `form:"server_id" json:"server_id" validate:"required|exists:database_servers,id"`
	Index    string `form:"index" json:"index" validate:"required"`
	ID       string `form:"id" json:"id"`
	Body     string `form:"body" json:"body" validate:"required"`
}

// DatabaseESDocumentDelete 删除文档
type DatabaseESDocumentDelete struct {
	ServerID uint   `form:"server_id" json:"server_id" validate:"required|exists:database_servers,id"`
	Index    string `form:"index" json:"index" validate:"required"`
	ID       string `form:"id" json:"id" validate:"required"`
}
