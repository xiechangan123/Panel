package openresty

import (
	"github.com/go-chi/chi/v5"

	"github.com/acepanel/panel/v3/internal/apps/nginx"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
)

type App struct {
	nginx *nginx.App
}

func NewApp(nginxApp *nginx.App) *App {
	return &App{
		nginx: nginxApp,
	}
}

func (s *App) Route(r chi.Router) {
	s.nginx.Route(r)
}

func (s *App) Status() string {
	ok, _ := systemctl.Status("nginx")
	return types.AggregateAppStatus(ok)
}
