package biz

import (
	"github.com/acepanel/panel/v3/pkg/api"
	"github.com/acepanel/panel/v3/pkg/types"
)

type TemplateRepo interface {
	List() api.Templates
	Get(slug string) (*api.Template, error)
	Callback(slug string) error
	CreateCompose(name, compose string, envs []types.KV, autoFirewall bool) (string, error)
}

type TemplateUsecase struct {
	repo TemplateRepo
}

func NewTemplateUsecase(repo TemplateRepo) *TemplateUsecase {
	return &TemplateUsecase{repo: repo}
}

func (uc *TemplateUsecase) List() api.Templates {
	return uc.repo.List()
}

func (uc *TemplateUsecase) Get(slug string) (*api.Template, error) {
	return uc.repo.Get(slug)
}

func (uc *TemplateUsecase) Callback(slug string) error {
	return uc.repo.Callback(slug)
}

func (uc *TemplateUsecase) CreateCompose(name, compose string, envs []types.KV, autoFirewall bool) (string, error) {
	return uc.repo.CreateCompose(name, compose, envs, autoFirewall)
}
