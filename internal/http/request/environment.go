package request

// EnvironmentAction 环境操作请求
type EnvironmentAction struct {
	Type string `json:"type"`
	Slug string `json:"slug"`
}
