package apploader

import (
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/acepanel/panel/v3/pkg/types"
)

var apps sync.Map

var supportedSlugs = []string{
	"apache",
	"clickhouse",
	"codeserver",
	"docker",
	"elasticsearch",
	"fail2ban",
	"frp",
	"gitea",
	"grafana",
	"kafka",
	"mariadb",
	"memcached",
	"minio",
	"mongodb",
	"mysql",
	"nginx",
	"openresty",
	"opensearch",
	"percona",
	"pgadmin",
	"phpmyadmin",
	"podman",
	"postgresql",
	"prometheus",
	"pureftpd",
	"redis",
	"rocketmq",
	"rsync",
	"s3fs",
	"supervisor",
	"valkey",
}

type Loader struct{}

func (r *Loader) Add(app ...types.App) {
	for item := range slices.Values(app) {
		slug := getSlug(item)
		apps.Store(slug, item)
	}
}

func (r *Loader) Register(mux chi.Router) {
	apps.Range(func(key, value any) bool {
		app := value.(types.App)
		mux.Route("/"+key.(string), app.Route)
		return true
	})
}

func (r *Loader) Get(slug string) (types.App, bool) {
	v, ok := apps.Load(slug)
	if !ok {
		return nil, false
	}
	return v.(types.App), true
}

func Slugs() []string {
	var slugs []string
	apps.Range(func(key, value any) bool {
		slugs = append(slugs, key.(string))
		return true
	})
	return slugs
}

// SupportedSlugs 返回当前版本内置的应用标识，无需构造应用实例。
func SupportedSlugs() []string {
	return slices.Clone(supportedSlugs)
}

func getSlug(app types.App) string {
	if app == nil {
		return ""
	}

	t := reflect.TypeOf(app)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	pkgPath := t.PkgPath()
	if pkgPath == "" {
		return ""
	}

	parts := strings.Split(pkgPath, "/")
	return parts[len(parts)-1]
}
