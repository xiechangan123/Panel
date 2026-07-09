package biz

import (
	"github.com/acepanel/panel/v3/pkg/api"
	"github.com/acepanel/panel/v3/pkg/types"
)

type EnvironmentRepo interface {
	Types() []types.LV
	All(typ ...string) api.Environments
	IsInstalled(typ, slug string) bool
	InstalledSlugs(typ string) []string
	InstalledVersion(typ, slug string) string
	HasUpdate(typ, slug string) bool
	Install(typ, slug string) error
	Uninstall(typ, slug string) error
	Update(typ, slug string) error
}

type EnvironmentUsecase struct {
	repo EnvironmentRepo
}

func NewEnvironmentUsecase(repo EnvironmentRepo) *EnvironmentUsecase {
	return &EnvironmentUsecase{repo: repo}
}

func (uc *EnvironmentUsecase) Types() []types.LV {
	return uc.repo.Types()
}

func (uc *EnvironmentUsecase) All(typ ...string) api.Environments {
	return uc.repo.All(typ...)
}

func (uc *EnvironmentUsecase) IsInstalled(typ, slug string) bool {
	return uc.repo.IsInstalled(typ, slug)
}

func (uc *EnvironmentUsecase) InstalledSlugs(typ string) []string {
	return uc.repo.InstalledSlugs(typ)
}

func (uc *EnvironmentUsecase) InstalledVersion(typ, slug string) string {
	return uc.repo.InstalledVersion(typ, slug)
}

func (uc *EnvironmentUsecase) HasUpdate(typ, slug string) bool {
	return uc.repo.HasUpdate(typ, slug)
}

func (uc *EnvironmentUsecase) Install(typ, slug string) error {
	return uc.repo.Install(typ, slug)
}

func (uc *EnvironmentUsecase) Uninstall(typ, slug string) error {
	return uc.repo.Uninstall(typ, slug)
}

func (uc *EnvironmentUsecase) Update(typ, slug string) error {
	return uc.repo.Update(typ, slug)
}
