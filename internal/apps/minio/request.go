package minio

type UpdateEnv struct {
	Env string `form:"env" json:"env" validate:"required"`
}
