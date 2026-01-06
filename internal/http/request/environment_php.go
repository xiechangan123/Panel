package request

type EnvironmentPHPVersion struct {
	Version uint `json:"version"`
}

type EnvironmentPHPModule struct {
	Version uint   `json:"version"`
	Slug    string `form:"slug" json:"slug" validate:"required"`
}

type EnvironmentPHPUpdateConfig struct {
	Version uint   `json:"version"`
	Config  string `form:"config" json:"config" validate:"required"`
}
