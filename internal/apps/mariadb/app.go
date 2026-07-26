package mariadb

import (
	"github.com/go-chi/chi/v5"

	"github.com/acepanel/panel/v3/internal/apps/mysql"
)

type App struct {
	mysql *mysql.App
}

func NewApp(mysqlApp *mysql.App) (*App, error) {
	return &App{
		mysql: mysqlApp,
	}, nil
}

func (s *App) Route(r chi.Router) {
	s.mysql.Route(r)
}

func (s *App) Status() string {
	return s.mysql.Status()
}
