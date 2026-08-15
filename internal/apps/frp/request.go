package frp

type Name struct {
	Name string `form:"name" json:"name" validate:"required && in:frps,frpc"`
}

type UpdateConfig struct {
	Name   string `form:"name" json:"name" validate:"required && in:frps,frpc"`
	Config string `form:"config" json:"config" validate:"required"`
}

type UpdateUser struct {
	Name  string `form:"name" json:"name" validate:"required && in:frps,frpc"`
	User  string `form:"user" json:"user" validate:"required && regex:\"^[a-zA-Z0-9_-]+$\""`
	Group string `form:"group" json:"group" validate:"required && regex:\"^[a-zA-Z0-9_-]+$\""`
}

// ItemName 代理或访问者的名称，来自路径参数
type ItemName struct {
	Name string `form:"name" json:"name" validate:"required && regex:\"^[a-zA-Z0-9_.-]+$\""`
}
