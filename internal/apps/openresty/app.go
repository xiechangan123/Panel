package openresty

import (
	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/v3/internal/apps/nginx"
	"github.com/acepanel/panel/v3/pkg/systemctl"
	"github.com/acepanel/panel/v3/pkg/types"
)

type App struct {
	nginx *nginx.App
}

func NewApp(t *gotext.Locale) *App {
	return &App{
		nginx: nginx.NewApp(t),
	}
}

func (s *App) Route(r chi.Router) {
	s.nginx.Route(r)
}

func (s *App) Status() string {
	ok, _ := systemctl.Status("openresty")
	return types.AggregateAppStatus(ok)
}
