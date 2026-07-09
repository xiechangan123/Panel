package openresty

import (
	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/apps/nginx"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
)

type App struct {
	nginx *nginx.App
}

func NewApp(i do.Injector) (*App, error) {
	return &App{
		nginx: do.MustInvoke[*nginx.App](i),
	}, nil
}

func (s *App) Route(r chi.Router) {
	s.nginx.Route(r)
}

func (s *App) Status() string {
	ok, _ := systemctl.Status("nginx")
	return types.AggregateAppStatus(ok)
}
