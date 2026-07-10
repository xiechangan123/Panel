package data

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/app"
	"github.com/acepanel/panel/v3/internal/biz"
	"github.com/acepanel/panel/v3/pkg/config"
	"github.com/acepanel/panel/v3/pkg/shell"
)

type environmentRepo struct {
	t    *gotext.Locale
	conf *config.Config
}

func NewEnvironmentRepo(i do.Injector) (biz.EnvironmentRepo, error) {
	return &environmentRepo{
		t:    do.MustInvoke[*gotext.Locale](i),
		conf: do.MustInvoke[*config.Config](i),
	}, nil
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
	case "dotnet":
		binFile = filepath.Join(path, "dotnet")
	default:
		return false
	}

	_, err := os.Stat(binFile)
	return err == nil
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
		// OpenJDK Runtime Environment Corretto-21.0.9.11.1 (build 21.0.9+11-LTS) -> 21.0.9.11.1
		version, err = shell.Exec(filepath.Join(basePath, "bin", "java") + ` -version 2>&1 | sed -n 's/.*Corretto-\([0-9.]*\).*/\1/p' | head -n 1`)
	case "nodejs":
		// v20.10.0 -> 20.10.0
		version, err = shell.Exec(filepath.Join(basePath, "bin", "node") + " -v | sed 's/v//'")
	case "php":
		// PHP 8.3.0 (cli) -> 8.3.0
		version, err = shell.Exec(filepath.Join(basePath, "bin", "php") + " -d error_reporting=0 -r 'echo PHP_VERSION;'")
	case "python":
		// Python 3.11.5 -> 3.11.5
		version, err = shell.Exec(filepath.Join(basePath, "bin", "python3") + " --version | awk '{print $2}'")
	case "dotnet":
		// 8.0.100
		version, err = shell.Exec(filepath.Join(basePath, "dotnet") + " --version")
	default:
		return ""
	}

	if err != nil {
		return ""
	}
	return version
}

func (r *environmentRepo) ScriptCommand(typ, action, slug, version string) string {
	shellUrl := fmt.Sprintf("https://%s/%s/%s.sh", r.conf.App.DownloadEndpoint, typ, action)
	return fmt.Sprintf(`curl -sSLm 10 --retry 3 "%s" | bash -s -- "%s" "%s"`, shellUrl, slug, version)
}

func (r *environmentRepo) ExecScript(cmd string) error {
	return shell.ExecfWithOutput(cmd)
}
