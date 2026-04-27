package middleware

import (
	"net/http"
	"strings"

	"github.com/leonelquinteros/gotext"
	"github.com/samber/lo"

	"github.com/acepanel/panel/v3/internal/biz"
)

// MustInstall 确保已安装应用
func MustInstall(t *gotext.Locale, app biz.AppRepo) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var slugs []string
			if strings.HasPrefix(r.URL.Path, "/api/website") {
				slugs = append(slugs, "nginx", "openresty", "apache", "openlitespeed", "caddy")
			} else if strings.HasPrefix(r.URL.Path, "/api/container") {
				slugs = append(slugs, "podman", "docker")
			} else if strings.HasPrefix(r.URL.Path, "/api/apps/") {
				pathArr := strings.Split(r.URL.Path, "/")
				if len(pathArr) < 4 {
					Abort(w, http.StatusForbidden, t.Get("app not found"))
					return
				}
				slugs = append(slugs, pathArr[3])
			}

			flag := lo.SomeBy(slugs, func(s string) bool {
				installed, _ := app.IsInstalled("slug = ?", s)
				return installed
			})
			if !flag && len(slugs) > 0 {
				Abort(w, http.StatusForbidden, t.Get("app %s not installed", slugs))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
