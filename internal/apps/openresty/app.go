package openresty

import (
	"github.com/go-chi/chi/v5"
	"github.com/leonelquinteros/gotext"

	"github.com/acepanel/panel/internal/apps/nginx"
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
