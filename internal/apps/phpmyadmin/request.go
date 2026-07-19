package phpmyadmin

type UpdateConfig struct {
	Config string `form:"config" json:"config" validate:"required"`
}

type UpdatePort struct {
	Port uint `form:"port" json:"port" validate:"required && number && min:1 && max:65535"`
}

type Login struct {
	ServerID uint `form:"server_id" json:"server_id" validate:"required"`
}
