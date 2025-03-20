package request

type ContainerComposeGet struct {
	Name string `uri:"name" validate:"required"`
}

type ContainerComposeCreate struct {
	Name    string `json:"name" validate:"required"`
	Compose string `json:"compose" validate:"required"`
	Env     string `json:"env"`
}

type ContainerComposeUpdate struct {
	Name    string `uri:"name" validate:"required"`
	Compose string `json:"compose" validate:"required"`
	Env     string `json:"env"`
}

type ContainerComposeUp struct {
	Name  string `uri:"name" validate:"required"`
	Force bool   `json:"force"`
}

type ContainerComposeDown struct {
	Name string `uri:"name" validate:"required"`
}

type ContainerComposeRemove struct {
	Name string `uri:"name" validate:"required"`
}
