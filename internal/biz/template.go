package biz

import (
	"github.com/acepanel/panel/pkg/api"
	"github.com/acepanel/panel/pkg/types"
)

type TemplateRepo interface {
	List() api.Templates
	Get(slug string) (*api.Template, error)
	Callback(slug string) error
	CreateCompose(name, compose string, envs []types.KV, autoFirewall bool) (string, error)
}
