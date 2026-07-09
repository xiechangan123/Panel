package percona

import (
	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/apps/mysql"
)

type App struct {
	mysql *mysql.App
}

func NewApp(i do.Injector) (*App, error) {
	return &App{
		mysql: do.MustInvoke[*mysql.App](i),
	}, nil
}

func (s *App) Route(r chi.Router) {
	s.mysql.Route(r)
}

func (s *App) Status() string {
	return s.mysql.Status()
}
