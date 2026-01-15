package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/internal/app"
	"github.com/acepanel/panel/internal/biz"
	"github.com/acepanel/panel/pkg/api"
	"github.com/acepanel/panel/pkg/config"
	"github.com/acepanel/panel/pkg/shell"
	"github.com/acepanel/panel/pkg/types"
)

type environmentRepo struct {
	t     *gotext.Locale
	conf  *config.Config
	cache biz.CacheRepo
	task  biz.TaskRepo
}

func NewEnvironmentRepo(t *gotext.Locale, conf *config.Config, cache biz.CacheRepo, task biz.TaskRepo) biz.EnvironmentRepo {
	return &environmentRepo{
		t:     t,
		conf:  conf,
		cache: cache,
		task:  task,
	}
}

func (r *environmentRepo) Types() []types.LV {
	return []types.LV{
		{Label: "Go", Value: "go"},
		{Label: "Java", Value: "java"},
		{Label: "Node.js", Value: "nodejs"},
		{Label: "PHP", Value: "php"},
		{Label: "Python", Value: "python"},
	}
}

func (r *environmentRepo) All(typ ...string) api.Environments {
	cached, err := r.cache.Get(biz.CacheKeyEnvironment)
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

func (r *environmentRepo) GetByTypeAndSlug(typ, slug string) *api.Environment {
	all := r.All()
	for _, env := range all {
		if env.Type == typ && env.Slug == slug {
			return env
		}
	}
	return nil
}

func (r *environmentRepo) IsInstalled(typ, slug string) bool {
	path := filepath.Join(app.Root, "server", typ, slug)
	var binFile string
	switch typ {
	case "go":
		binFile = filepath.Join(path, "bin", "go")
	case "java":
		binFile = filepath.Join(path, "bin", "java")
	case "nodejs":
		binFile = filepath.Join(path, "bin", "node")
	case "php":
		binFile = filepath.Join(path, "bin", "php")
	case "python":
		binFile = filepath.Join(path, "bin", "python3")
	default:
		return false
	}

	_, err := os.Stat(binFile)
	return err == nil
}

func (r *environmentRepo) InstalledSlugs(typ string) []string {
	var slugs []string
	all := r.All()
	for _, env := range all {
		if env.Type != typ {
			continue
		}
		if r.IsInstalled(typ, env.Slug) {
			slugs = append(slugs, env.Slug)
		}
	}
	return slugs
}

func (r *environmentRepo) InstalledVersion(typ, slug string) string {
	if !r.IsInstalled(typ, slug) {
		return ""
	}

	var basePath = filepath.Join(app.Root, "server", typ, slug)
	var version string
	var err error

	switch typ {
	case "go":
		// go version go1.21.0 linux/amd64 -> 1.21.0
		version, err = shell.Exec(filepath.Join(basePath, "bin", "go") + " version | awk '{print $3}' | sed 's/go//'")
	case "java":
		// openjdk version "17.0.8" 2023-07-18 LTS -> 17.0.8
		version, err = shell.Exec(filepath.Join(basePath, "bin", "java") + " -version 2>&1 | head -n 1 | awk -F'\"' '{print $2}'")
	case "nodejs":
		// v20.10.0 -> 20.10.0
		version, err = shell.Exec(filepath.Join(basePath, "bin", "node") + " -v | sed 's/v//'")
	case "php":
		// PHP 8.3.0 (cli) -> 8.3.0
		version, err = shell.Exec(filepath.Join(basePath, "bin", "php") + " -v | head -n 1 | awk '{print $2}'")
	case "python":
		// Python 3.11.5 -> 3.11.5
		version, err = shell.Exec(filepath.Join(basePath, "bin", "python3") + " --version | awk '{print $2}'")
	default:
		return ""
	}

	if err != nil {
		return ""
	}
	return version
}

func (r *environmentRepo) HasUpdate(typ, slug string) bool {
	if !r.IsInstalled(typ, slug) {
		return false
	}
	env := r.GetByTypeAndSlug(typ, slug)
	if env == nil {
		return false
	}

	mainlineVersion := env.Version
	installedVersion := r.InstalledVersion(typ, slug)

	return mainlineVersion != installedVersion && mainlineVersion != "" && installedVersion != ""
}

func (r *environmentRepo) Install(typ, slug string) error {
	if installed := r.IsInstalled(typ, slug); installed {
		return errors.New(r.t.Get("environment %s-%s already installed", typ, slug))
	}
	return r.do(typ, slug, "install")
}

func (r *environmentRepo) Uninstall(typ, slug string) error {
	if installed := r.IsInstalled(typ, slug); !installed {
		return errors.New(r.t.Get("environment %s-%s not installed", typ, slug))
	}
	return r.do(typ, slug, "uninstall")
}

func (r *environmentRepo) Update(typ, slug string) error {
	if installed := r.IsInstalled(typ, slug); !installed {
		return errors.New(r.t.Get("environment %s-%s not installed", typ, slug))
	}
	return r.do(typ, slug, "update")
}

func (r *environmentRepo) do(typ, slug, action string) error {
	env := r.GetByTypeAndSlug(typ, slug)
	if env == nil {
		return fmt.Errorf("environment not found: %s-%s", typ, slug)
	}

	shellUrl := fmt.Sprintf("https://%s/%s/%s.sh", r.conf.App.DownloadEndpoint, typ, action)

	if app.IsCli {
		return shell.ExecfWithOutput(`curl -sSLm 10 --retry 3 "%s" | bash -s -- "%s" "%s"`, shellUrl, env.Slug, env.Version)
	}

	var name string
	switch action {
	case "install":
		name = r.t.Get("Install environment %s", env.Name)
	case "uninstall":
		name = r.t.Get("Uninstall environment %s", env.Name)
	case "update":
		name = r.t.Get("Update environment %s", env.Name)
	}

	task := new(biz.Task)
	task.Name = name
	task.Status = biz.TaskStatusWaiting
	task.Shell = fmt.Sprintf(`curl -sSLm 10 --retry 3 "%s" | bash -s -- "%s" "%s" >> /tmp/%s-%s.log 2>&1`, shellUrl, env.Slug, env.Version, env.Type, env.Slug)
	task.Log = fmt.Sprintf("/tmp/%s-%s.log", env.Type, env.Slug)

	return r.task.Push(task)
}
