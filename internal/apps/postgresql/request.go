package postgresql

type UpdateConfig struct {
	Config string `form:"config" json:"config" validate:"required"`
}

type SetPostgresPassword struct {
	Password string `form:"password" json:"password" validate:"required|password"`
}
