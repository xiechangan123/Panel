package types

// RedirectType 重定向类型
type RedirectType string

const (
	RedirectType404  RedirectType = "404"  // 404 重定向
	RedirectTypeHost RedirectType = "host" // 主机名重定向
	RedirectTypeURL  RedirectType = "url"  // URL 重定向
)

// Redirect 重定向配置
type Redirect struct {
	Type       RedirectType `json:"type"`        // 重定向类型
	From       string       `json:"from"`        // 源地址，如: "example.com", "http://example.com", "/old"
	To         string       `json:"to"`          // 目标地址，如: "https://example.com"
	KeepURI    bool         `json:"keep_uri"`    // 是否保持 URI 不变（即保留请求参数）
	StatusCode int          `json:"status_code"` // 自定义状态码，如: 301, 302, 307, 308，默认 308
}
