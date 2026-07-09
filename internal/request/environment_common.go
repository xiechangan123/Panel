package request

// EnvironmentSlug 环境版本请求（通用）
type EnvironmentSlug struct {
	Slug string `json:"slug"`
}

// EnvironmentProxy Go 代理设置请求
type EnvironmentProxy struct {
	Slug  string `json:"slug"`
	Proxy string `form:"proxy" json:"proxy" validate:"required"`
}

// EnvironmentRegistry Node.js 镜像设置请求
type EnvironmentRegistry struct {
	Slug     string `json:"slug"`
	Registry string `form:"registry" json:"registry" validate:"required"`
}

// EnvironmentMirror Python 镜像设置请求
type EnvironmentMirror struct {
	Slug   string `json:"slug"`
	Mirror string `form:"mirror" json:"mirror" validate:"required"`
}
