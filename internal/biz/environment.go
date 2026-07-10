package biz

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/pkg/api"
	"github.com/acepanel/panel/v3/pkg/types"
)

type EnvironmentRepo interface {
	IsInstalled(typ, slug string) bool
	InstalledVersion(typ, slug string) string
	ScriptCommand(typ, action, slug, version string) string
	ExecScript(cmd string) error
}

type EnvironmentUsecase struct {
	repo  EnvironmentRepo
	cache CacheRepo
	task  TaskRepo
	t     *gotext.Locale
}

func NewEnvironmentUsecase(i do.Injector) (*EnvironmentUsecase, error) {
	return &EnvironmentUsecase{
		repo:  do.MustInvoke[EnvironmentRepo](i),
		cache: do.MustInvoke[CacheRepo](i),
		task:  do.MustInvoke[TaskRepo](i),
		t:     do.MustInvoke[*gotext.Locale](i),
	}, nil
}

func (uc *EnvironmentUsecase) Types() []types.LV {
	return []types.LV{
		{Label: "Go", Value: "go"},
		{Label: "Java", Value: "java"},
		{Label: "Node.js", Value: "nodejs"},
		{Label: "PHP", Value: "php"},
		{Label: "Python", Value: "python"},
		{Label: ".NET", Value: "dotnet"},
	}
}

func (uc *EnvironmentUsecase) All(typ ...string) api.Environments {
	cached, err := uc.cache.Get(CacheKeyEnvironment)
	if err != nil {
		return nil
	}
	var environments api.Environments
	if err = json.Unmarshal([]byte(cached), &environments); err != nil {
		return nil
	}

	// 过滤
	environments = slices.DeleteFunc(environments, func(env *api.Environment) bool {
		return len(typ) > 0 && typ[0] != "" && env.Type != typ[0]
	})

	return environments
}

func (uc *EnvironmentUsecase) getByTypeAndSlug(typ, slug string) *api.Environment {
	all := uc.All()
	for _, env := range all {
		if env.Type == typ && env.Slug == slug {
			return env
		}
	}
	return nil
}

func (uc *EnvironmentUsecase) IsInstalled(typ, slug string) bool {
	return uc.repo.IsInstalled(typ, slug)
}

func (uc *EnvironmentUsecase) InstalledSlugs(typ string) []string {
	var slugs []string
	all := uc.All()
	for _, env := range all {
		if env.Type != typ {
			continue
		}
		if uc.repo.IsInstalled(typ, env.Slug) {
			slugs = append(slugs, env.Slug)
		}
	}
	return slugs
}

func (uc *EnvironmentUsecase) InstalledVersion(typ, slug string) string {
	return uc.repo.InstalledVersion(typ, slug)
}

func (uc *EnvironmentUsecase) HasUpdate(typ, slug string) bool {
	if !uc.repo.IsInstalled(typ, slug) {
		return false
	}
	env := uc.getByTypeAndSlug(typ, slug)
	if env == nil {
		return false
	}

	mainlineVersion := env.Version
	installedVersion := uc.repo.InstalledVersion(typ, slug)

	return mainlineVersion != installedVersion && mainlineVersion != "" && installedVersion != ""
}

func (uc *EnvironmentUsecase) Install(typ, slug string) error {
	if installed := uc.repo.IsInstalled(typ, slug); installed {
		return errors.New(uc.t.Get("environment %s-%s already installed", typ, slug))
	}
	return uc.do(typ, slug, "install")
}

func (uc *EnvironmentUsecase) Uninstall(typ, slug string) error {
	if installed := uc.repo.IsInstalled(typ, slug); !installed {
		return errors.New(uc.t.Get("environment %s-%s not installed", typ, slug))
	}
	return uc.do(typ, slug, "uninstall")
}

func (uc *EnvironmentUsecase) Update(typ, slug string) error {
	if installed := uc.repo.IsInstalled(typ, slug); !installed {
		return errors.New(uc.t.Get("environment %s-%s not installed", typ, slug))
	}
	return uc.do(typ, slug, "update")
}

func (uc *EnvironmentUsecase) do(typ, slug, action string) error {
	env := uc.getByTypeAndSlug(typ, slug)
	if env == nil {
		return fmt.Errorf("environment not found: %s-%s", typ, slug)
	}

	cmd := uc.repo.ScriptCommand(typ, action, env.Slug, env.Version)

	if app.IsCli {
		return uc.repo.ExecScript(cmd)
	}

	var name string
	switch action {
	case "install":
		name = uc.t.Get("Install environment %s", env.Name)
	case "uninstall":
		name = uc.t.Get("Uninstall environment %s", env.Name)
	case "update":
		name = uc.t.Get("Update environment %s", env.Name)
	}

	task := new(Task)
	task.Name = name
	task.Status = TaskStatusWaiting
	task.Shell = cmd

	return uc.task.Push(task)
}
