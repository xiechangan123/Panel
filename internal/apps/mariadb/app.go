package mariadb

import (
	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"

	"github.com/acepanel/panel/v3/internal/apps/mysql"
)

type App struct {
	mysql *mysql.App
}

func NewApp(i do.Injector) (*App, error) {
	app, err := mysql.NewApp(i)
	if err != nil {
		return nil, err
	}
	return &App{
		mysql: app,
	}, nil
}

func (s *App) Route(r chi.Router) {
	s.mysql.Route(r)
}

func (s *App) Status() string {
	return s.mysql.Status()
}
