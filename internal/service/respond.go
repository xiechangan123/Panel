package service

// Envelope 泛型响应信封，仅用于 OpenAPI 文档样本。
type Envelope[T any] struct {
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

// Page 泛型列表载荷，仅用于 OpenAPI 文档样本。
type Page[T any] struct {
	Total int64 `json:"total"`
	Items []T   `json:"items"`
}
