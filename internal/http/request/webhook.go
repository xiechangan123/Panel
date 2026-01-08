package request

type WebHookCreate struct {
	Name   string `json:"name" form:"name" validate:"required"`
	Script string `json:"script" form:"script" validate:"required"`
	Raw    bool   `json:"raw" form:"raw"`
	User   string `json:"user" form:"user"`
}

type WebHookUpdate struct {
	ID     uint   `json:"id" form:"id" uri:"id" validate:"required|exists:web_hooks,id"`
	Name   string `json:"name" form:"name" validate:"required"`
	Script string `json:"script" form:"script" validate:"required"`
	Raw    bool   `json:"raw" form:"raw"`
	User   string `json:"user" form:"user" validate:"required"`
	Status bool   `json:"status" form:"status"`
}

type WebHookKey struct {
	Key string `json:"key" form:"key" uri:"key" validate:"required"`
}
