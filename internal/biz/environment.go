package biz

import (
	"github.com/acepanel/panel/pkg/api"
	"github.com/acepanel/panel/pkg/types"
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
