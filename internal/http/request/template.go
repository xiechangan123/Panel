package request

import "github.com/acepanel/panel/pkg/types"

type TemplateSlug struct {
	Slug string `uri:"slug" validate:"required"`
}

type TemplateCreate struct {
	Slug         string     `json:"slug" validate:"required"`
	Name         string     `json:"name" validate:"required|regex:^[a-zA-Z0-9_-]+$"`
	Compose      string     `json:"compose"`
	Envs         []types.KV `json:"envs"`
	AutoFirewall bool       `json:"auto_firewall"`
}
