package middleware

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"

	"github.com/tnb-labs/panel/internal/biz"
)

// MustInstall 确保已安装应用
func MustInstall(t *gotext.Locale, app biz.AppRepo) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			routePath := chi.RouteContext(r.Context()).RoutePath

			var slugs []string
			if strings.HasPrefix(routePath, "/api/website") {
				slugs = append(slugs, "nginx")
			} else if strings.HasPrefix(routePath, "/api/container") {
				slugs = append(slugs, "podman", "docker")
			} else if strings.HasPrefix(routePath, "/api/apps/") {
				pathArr := strings.Split(routePath, "/")
				if len(pathArr) < 4 {
					Abort(w, http.StatusForbidden, t.Get("app not found"))
					return
				}
				slugs = append(slugs, pathArr[3])
			}

			flag := false
			for _, s := range slugs {
				if installed, _ := app.IsInstalled("slug = ?", s); installed {
					flag = true
					break
				}
			}
			if !flag && len(slugs) > 0 {
				Abort(w, http.StatusForbidden, t.Get("app %s not installed", slugs))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
